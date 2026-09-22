package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/juho05/sheetopia-sync/config"
	"github.com/juho05/sheetopia-sync/database"
)

const (
	testUser    = "alice"
	testAuthKey = "test-auth-key"
)

func setupHandler(t *testing.T) (context.Context, *database.Queries, *Handler) {
	t.Helper()
	ctx := context.Background()
	config.DataDir = t.TempDir()

	db, queries, err := database.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %s", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	err = queries.CreateUser(ctx, database.CreateUserParams{Name: testUser, PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("create user: %s", err)
	}
	err = queries.CreateAuthKey(ctx, database.CreateAuthKeyParams{
		KeyHash: database.HashAuthKey(testAuthKey),
		User:    testUser,
	})
	if err != nil {
		t.Fatalf("create auth key: %s", err)
	}

	return ctx, queries, NewHandler(db, queries)
}

func request(t *testing.T, h *Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %s", err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}

	r := httptest.NewRequest(method, path, reader)
	r.Header.Set("Authorization", "Bearer "+testAuthKey)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func decodeResponse[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	err := json.Unmarshal(w.Body.Bytes(), &v)
	if err != nil {
		t.Fatalf("decode response %q: %s", w.Body.String(), err)
	}
	return v
}

type scoreRequest struct {
	Title             string          `json:"title"`
	MetadataUpdatedAt time.Time       `json:"metadataUpdatedAt"`
	WrittenAt         *time.Time      `json:"writtenAt,omitempty"`
	Metadata          json.RawMessage `json:"metadata"`
	TagIDs            []string        `json:"tagIds"`
}

type tagRequest struct {
	Name      string     `json:"name"`
	Color     int        `json:"color"`
	Type      string     `json:"type,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
	WrittenAt *time.Time `json:"writtenAt,omitempty"`
}

type setlistRequest struct {
	Name      string     `json:"name"`
	ScoreIDs  []string   `json:"scoreIds"`
	UpdatedAt time.Time  `json:"updatedAt"`
	WrittenAt *time.Time `json:"writtenAt,omitempty"`
}

func newScoreRequest(writtenAt *time.Time) scoreRequest {
	return scoreRequest{
		Title:             "score",
		MetadataUpdatedAt: time.Unix(1000, 0),
		WrittenAt:         writtenAt,
		Metadata:          json.RawMessage(`{}`),
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestResurrectScore(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/score/score-1", newScoreRequest(ptr(time.Now())))
	if w.Code != http.StatusOK {
		t.Fatalf("initial upsert returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[struct {
		HasFile bool `json:"hasFile"`
	}](t, w)
	if body.HasFile {
		t.Error("initial upsert reported hasFile true")
	}

	// a score the server holds a file for reports hasFile true on the next metadata write
	_, err := q.UpdateFileInfo(ctx, database.UpdateFileInfoParams{
		FileUpdatedAt:   time.Unix(1000, 0),
		FileUpdatedAt_2: time.Unix(1000, 0),
		FileType:        "pdf",
		User:            testUser,
		ID:              "score-1",
	})
	if err != nil {
		t.Fatalf("update file info: %s", err)
	}
	req := newScoreRequest(ptr(time.Now()))
	req.MetadataUpdatedAt = time.Unix(2000, 0)
	w = request(t, h, http.MethodPost, "/api/score/score-1", req)
	if w.Code != http.StatusOK {
		t.Fatalf("second upsert returned %d: %s", w.Code, w.Body.String())
	}
	body = decodeResponse[struct {
		HasFile bool `json:"hasFile"`
	}](t, w)
	if !body.HasFile {
		t.Error("upsert of a score with a file reported hasFile false")
	}

	w = request(t, h, http.MethodDelete, "/api/score/score-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	marker, err := q.FindDeletedScoreMarker(ctx, database.FindDeletedScoreMarkerParams{
		User:    testUser,
		ScoreID: "score-1",
	})
	if err != nil {
		t.Fatalf("find deleted score marker: %s", err)
	}

	// a write from before the deletion loses
	w = request(t, h, http.MethodPost, "/api/score/score-1", newScoreRequest(ptr(marker.DeletedAt.Add(-time.Hour))))
	if w.Code != http.StatusConflict {
		t.Fatalf("stale write returned %d, want 409: %s", w.Code, w.Body.String())
	}
	conflict := decodeResponse[struct {
		DeletedAt time.Time `json:"deletedAt"`
	}](t, w)
	if !conflict.DeletedAt.Equal(marker.DeletedAt) {
		t.Errorf("conflict reported deletedAt %s, want %s", conflict.DeletedAt, marker.DeletedAt)
	}
	_, err = q.FindDeletedScoreMarker(ctx, database.FindDeletedScoreMarkerParams{
		User:    testUser,
		ScoreID: "score-1",
	})
	if err != nil {
		t.Errorf("rejected write removed the marker: %s", err)
	}

	// a write from after the deletion re-creates the score and drops the marker
	w = request(t, h, http.MethodPost, "/api/score/score-1", newScoreRequest(ptr(marker.DeletedAt.Add(time.Hour))))
	if w.Code != http.StatusOK {
		t.Fatalf("resurrection returned %d, want 200: %s", w.Code, w.Body.String())
	}
	body = decodeResponse[struct {
		HasFile bool `json:"hasFile"`
	}](t, w)
	if body.HasFile {
		t.Error("resurrection reported hasFile true although the file was deleted with the score")
	}
	_, err = q.FindDeletedScoreMarker(ctx, database.FindDeletedScoreMarkerParams{
		User:    testUser,
		ScoreID: "score-1",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("marker survived the resurrection: %v", err)
	}
	score, err := q.FindScore(ctx, database.FindScoreParams{User: testUser, ID: "score-1"})
	if err != nil {
		t.Fatalf("find resurrected score: %s", err)
	}
	if score.FileType != "none" {
		t.Errorf("resurrected score has file type %q, want none", score.FileType)
	}
}

func TestResurrectTag(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		Color:     1,
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(time.Now()),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("initial upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodDelete, "/api/tag/tag-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	marker, err := q.FindDeletedTagMarker(ctx, database.FindDeletedTagMarkerParams{User: testUser, TagID: "tag-1"})
	if err != nil {
		t.Fatalf("find deleted tag marker: %s", err)
	}

	w = request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		Color:     1,
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(marker.DeletedAt.Add(-time.Hour)),
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("stale write returned %d, want 409: %s", w.Code, w.Body.String())
	}
	conflict := decodeResponse[struct {
		DeletedAt time.Time `json:"deletedAt"`
	}](t, w)
	if !conflict.DeletedAt.Equal(marker.DeletedAt) {
		t.Errorf("conflict reported deletedAt %s, want %s", conflict.DeletedAt, marker.DeletedAt)
	}
	_, err = q.FindDeletedTagMarker(ctx, database.FindDeletedTagMarkerParams{User: testUser, TagID: "tag-1"})
	if err != nil {
		t.Errorf("rejected write removed the marker: %s", err)
	}

	w = request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		Color:     1,
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(marker.DeletedAt.Add(time.Hour)),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("resurrection returned %d, want 200: %s", w.Code, w.Body.String())
	}
	_, err = q.FindDeletedTagMarker(ctx, database.FindDeletedTagMarkerParams{User: testUser, TagID: "tag-1"})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("marker survived the resurrection: %v", err)
	}
	_, err = q.FindTag(ctx, database.FindTagParams{User: testUser, ID: "tag-1"})
	if err != nil {
		t.Errorf("find resurrected tag: %s", err)
	}
}

func TestResurrectSetlist(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/setlist/setlist-1", setlistRequest{
		Name:      "setlist",
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(time.Now()),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("initial upsert returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodDelete, "/api/setlist/setlist-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	marker, err := q.FindDeletedSetlistMarker(ctx, database.FindDeletedSetlistMarkerParams{
		User:      testUser,
		SetlistID: "setlist-1",
	})
	if err != nil {
		t.Fatalf("find deleted setlist marker: %s", err)
	}

	w = request(t, h, http.MethodPost, "/api/setlist/setlist-1", setlistRequest{
		Name:      "setlist",
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(marker.DeletedAt.Add(-time.Hour)),
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("stale write returned %d, want 409: %s", w.Code, w.Body.String())
	}
	conflict := decodeResponse[struct {
		DeletedAt time.Time `json:"deletedAt"`
	}](t, w)
	if !conflict.DeletedAt.Equal(marker.DeletedAt) {
		t.Errorf("conflict reported deletedAt %s, want %s", conflict.DeletedAt, marker.DeletedAt)
	}
	_, err = q.FindDeletedSetlistMarker(ctx, database.FindDeletedSetlistMarkerParams{
		User:      testUser,
		SetlistID: "setlist-1",
	})
	if err != nil {
		t.Errorf("rejected write removed the marker: %s", err)
	}

	w = request(t, h, http.MethodPost, "/api/setlist/setlist-1", setlistRequest{
		Name:      "setlist",
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(marker.DeletedAt.Add(time.Hour)),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("resurrection returned %d, want 200: %s", w.Code, w.Body.String())
	}
	_, err = q.FindDeletedSetlistMarker(ctx, database.FindDeletedSetlistMarkerParams{
		User:      testUser,
		SetlistID: "setlist-1",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("marker survived the resurrection: %v", err)
	}
	_, err = q.FindSetlist(ctx, database.FindSetlistParams{User: testUser, ID: "setlist-1"})
	if err != nil {
		t.Errorf("find resurrected setlist: %s", err)
	}
}

func TestResurrectPracticeRecord(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/practice/record/record-1", newPracticeRecordRequest(time.Unix(1000, 0)))
	if w.Code != http.StatusOK {
		t.Fatalf("initial upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/practice/record/record-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	marker, err := q.FindDeletedPracticeRecordMarker(ctx, database.FindDeletedPracticeRecordMarkerParams{
		User:     testUser,
		RecordID: "record-1",
	})
	if err != nil {
		t.Fatalf("find deleted practice record marker: %s", err)
	}

	req := newPracticeRecordRequest(time.Unix(2000, 0))
	w = request(t, h, http.MethodPost, "/api/practice/record/record-1", req)
	if w.Code != http.StatusConflict {
		t.Fatalf("write without writtenAt returned %d, want 409: %s", w.Code, w.Body.String())
	}

	req.WrittenAt = ptr(marker.DeletedAt.Add(-time.Hour))
	w = request(t, h, http.MethodPost, "/api/practice/record/record-1", req)
	if w.Code != http.StatusConflict {
		t.Fatalf("stale write returned %d, want 409: %s", w.Code, w.Body.String())
	}

	req.WrittenAt = ptr(marker.DeletedAt.Add(time.Hour))
	w = request(t, h, http.MethodPost, "/api/practice/record/record-1", req)
	if w.Code != http.StatusOK {
		t.Fatalf("resurrection returned %d, want 200: %s", w.Code, w.Body.String())
	}
	_, err = q.FindDeletedPracticeRecordMarker(ctx, database.FindDeletedPracticeRecordMarkerParams{
		User:     testUser,
		RecordID: "record-1",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("marker survived the resurrection: %v", err)
	}
	_, err = q.FindPracticeRecord(ctx, database.FindPracticeRecordParams{User: testUser, ID: "record-1"})
	if err != nil {
		t.Errorf("find resurrected practice record: %s", err)
	}
}

// clients before API 0.3 do not send writtenAt and cannot complete a resurrection, so for them the
// deletion keeps winning like it did before resurrection existed, no matter how new the content is
func TestWriteWithoutWrittenAtNeverResurrects(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/score/score-1", newScoreRequest(nil))
	if w.Code != http.StatusOK {
		t.Fatalf("initial upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/score/score-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	marker, err := q.FindDeletedScoreMarker(ctx, database.FindDeletedScoreMarkerParams{
		User:    testUser,
		ScoreID: "score-1",
	})
	if err != nil {
		t.Fatalf("find deleted score marker: %s", err)
	}

	// content far newer than the deletion still loses without writtenAt
	req := newScoreRequest(nil)
	req.MetadataUpdatedAt = marker.DeletedAt.Add(time.Hour)
	w = request(t, h, http.MethodPost, "/api/score/score-1", req)
	if w.Code != http.StatusConflict {
		t.Fatalf("write without writtenAt returned %d, want 409: %s", w.Code, w.Body.String())
	}
	_, err = q.FindDeletedScoreMarker(ctx, database.FindDeletedScoreMarkerParams{
		User:    testUser,
		ScoreID: "score-1",
	})
	if err != nil {
		t.Errorf("rejected write removed the marker: %s", err)
	}
	_, err = q.FindScore(ctx, database.FindScoreParams{User: testUser, ID: "score-1"})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("rejected write re-created the score: %v", err)
	}

	// the same for tags
	w = request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{Name: "tag", UpdatedAt: time.Unix(1000, 0)})
	if w.Code != http.StatusOK {
		t.Fatalf("initial tag upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/tag/tag-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete tag returned %d: %s", w.Code, w.Body.String())
	}
	tagMarker, err := q.FindDeletedTagMarker(ctx, database.FindDeletedTagMarkerParams{User: testUser, TagID: "tag-1"})
	if err != nil {
		t.Fatalf("find deleted tag marker: %s", err)
	}

	w = request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		UpdatedAt: tagMarker.DeletedAt.Add(time.Hour),
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("tag write without writtenAt returned %d, want 409: %s", w.Code, w.Body.String())
	}
	_, err = q.FindTag(ctx, database.FindTagParams{User: testUser, ID: "tag-1"})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("rejected write re-created the tag: %v", err)
	}

	// and for setlists
	w = request(t, h, http.MethodPost, "/api/setlist/setlist-1", setlistRequest{Name: "setlist", UpdatedAt: time.Unix(1000, 0)})
	if w.Code != http.StatusOK {
		t.Fatalf("initial setlist upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/setlist/setlist-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete setlist returned %d: %s", w.Code, w.Body.String())
	}
	setlistMarker, err := q.FindDeletedSetlistMarker(ctx, database.FindDeletedSetlistMarkerParams{
		User:      testUser,
		SetlistID: "setlist-1",
	})
	if err != nil {
		t.Fatalf("find deleted setlist marker: %s", err)
	}

	w = request(t, h, http.MethodPost, "/api/setlist/setlist-1", setlistRequest{
		Name:      "setlist",
		UpdatedAt: setlistMarker.DeletedAt.Add(time.Hour),
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("setlist write without writtenAt returned %d, want 409: %s", w.Code, w.Body.String())
	}
	_, err = q.FindSetlist(ctx, database.FindSetlistParams{User: testUser, ID: "setlist-1"})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("rejected write re-created the setlist: %v", err)
	}
}

func TestKnownTagIDIsAssigned(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(time.Now()),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("tag upsert returned %d: %s", w.Code, w.Body.String())
	}

	req := newScoreRequest(ptr(time.Now()))
	req.TagIDs = []string{"tag-1"}
	w = request(t, h, http.MethodPost, "/api/score/score-1", req)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert with a tag id returned %d, want 200: %s", w.Code, w.Body.String())
	}

	tagIDs, err := q.GetAssignedTagIDs(ctx, database.GetAssignedTagIDsParams{User: testUser, ScoreID: "score-1"})
	if err != nil {
		t.Fatalf("get assigned tag ids: %s", err)
	}
	if !slices.Equal(tagIDs, []string{"tag-1"}) {
		t.Errorf("assigned tag ids are %v, want [tag-1]", tagIDs)
	}
}

func TestUnknownTagIDIsRejected(t *testing.T) {
	ctx, q, h := setupHandler(t)

	req := newScoreRequest(ptr(time.Now()))
	req.TagIDs = []string{"unknown-tag"}
	w := request(t, h, http.MethodPost, "/api/score/score-1", req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("upsert with an unknown tag id returned %d, want 400: %s", w.Code, w.Body.String())
	}

	_, err := q.FindScore(ctx, database.FindScoreParams{User: testUser, ID: "score-1"})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("rejected upsert created the score: %v", err)
	}
}
