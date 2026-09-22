package handlers

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/juho05/sheetopia-sync/database"
)

type deletedItemResponse struct {
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deletedAt"`
}

type deletedScoresResponse struct {
	ScoreIDs      []string              `json:"scoreIds"`
	DeletedScores []deletedItemResponse `json:"deletedScores"`
}

type deletedTagsResponse struct {
	TagIDs      []string              `json:"tagIds"`
	DeletedTags []deletedItemResponse `json:"deletedTags"`
}

type deletedSetlistsResponse struct {
	SetlistIDs      []string              `json:"setlistIds"`
	DeletedSetlists []deletedItemResponse `json:"deletedSetlists"`
}

func TestDeletedScoresReportDeletedAt(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/score/score-1", newScoreRequest(ptr(time.Now())))
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/score/score-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	marker, err := q.FindDeletedScoreMarker(ctx, database.FindDeletedScoreMarkerParams{User: testUser, ScoreID: "score-1"})
	if err != nil {
		t.Fatalf("find deleted score marker: %s", err)
	}

	w = request(t, h, http.MethodGet, "/api/score/deleted", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("deleted scores returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[deletedScoresResponse](t, w)

	if !slices.Equal(body.ScoreIDs, []string{"score-1"}) {
		t.Errorf("scoreIds are %v, want [score-1]", body.ScoreIDs)
	}
	if len(body.DeletedScores) != 1 {
		t.Fatalf("deletedScores are %v, want one entry", body.DeletedScores)
	}
	if body.DeletedScores[0].ID != "score-1" {
		t.Errorf("deletedScores[0].id is %q, want score-1", body.DeletedScores[0].ID)
	}
	if !body.DeletedScores[0].DeletedAt.Equal(marker.DeletedAt) {
		t.Errorf("deletedScores[0].deletedAt is %s, want %s", body.DeletedScores[0].DeletedAt, marker.DeletedAt)
	}
}

func TestDeletedTagsReportDeletedAt(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/tag/tag-1", tagRequest{
		Name:      "tag",
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(time.Now()),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/tag/tag-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	marker, err := q.FindDeletedTagMarker(ctx, database.FindDeletedTagMarkerParams{User: testUser, TagID: "tag-1"})
	if err != nil {
		t.Fatalf("find deleted tag marker: %s", err)
	}

	w = request(t, h, http.MethodGet, "/api/tag/deleted", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("deleted tags returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[deletedTagsResponse](t, w)

	if !slices.Equal(body.TagIDs, []string{"tag-1"}) {
		t.Errorf("tagIds are %v, want [tag-1]", body.TagIDs)
	}
	if len(body.DeletedTags) != 1 {
		t.Fatalf("deletedTags are %v, want one entry", body.DeletedTags)
	}
	if body.DeletedTags[0].ID != "tag-1" {
		t.Errorf("deletedTags[0].id is %q, want tag-1", body.DeletedTags[0].ID)
	}
	if !body.DeletedTags[0].DeletedAt.Equal(marker.DeletedAt) {
		t.Errorf("deletedTags[0].deletedAt is %s, want %s", body.DeletedTags[0].DeletedAt, marker.DeletedAt)
	}
}

func TestDeletedSetlistsReportDeletedAt(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/setlist/setlist-1", setlistRequest{
		Name:      "setlist",
		UpdatedAt: time.Unix(1000, 0),
		WrittenAt: ptr(time.Now()),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/setlist/setlist-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	marker, err := q.FindDeletedSetlistMarker(ctx, database.FindDeletedSetlistMarkerParams{User: testUser, SetlistID: "setlist-1"})
	if err != nil {
		t.Fatalf("find deleted setlist marker: %s", err)
	}

	w = request(t, h, http.MethodGet, "/api/setlist/deleted", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("deleted setlists returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[deletedSetlistsResponse](t, w)

	if !slices.Equal(body.SetlistIDs, []string{"setlist-1"}) {
		t.Errorf("setlistIds are %v, want [setlist-1]", body.SetlistIDs)
	}
	if len(body.DeletedSetlists) != 1 {
		t.Fatalf("deletedSetlists are %v, want one entry", body.DeletedSetlists)
	}
	if body.DeletedSetlists[0].ID != "setlist-1" {
		t.Errorf("deletedSetlists[0].id is %q, want setlist-1", body.DeletedSetlists[0].ID)
	}
	if !body.DeletedSetlists[0].DeletedAt.Equal(marker.DeletedAt) {
		t.Errorf("deletedSetlists[0].deletedAt is %s, want %s", body.DeletedSetlists[0].DeletedAt, marker.DeletedAt)
	}
}

// the deletedAt a client reads from the deleted endpoint is exactly the value that decides the
// resurrection, so a writtenAt one second later drops the marker and the id disappears from the list
func TestDeletedAtDrivesResurrection(t *testing.T) {
	_, _, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/score/score-1", newScoreRequest(ptr(time.Now())))
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/score/score-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/score/deleted", nil)
	body := decodeResponse[deletedScoresResponse](t, w)
	if len(body.DeletedScores) != 1 {
		t.Fatalf("deletedScores are %v, want one entry", body.DeletedScores)
	}
	deletedAt := body.DeletedScores[0].DeletedAt

	w = request(t, h, http.MethodPost, "/api/score/score-1", newScoreRequest(ptr(deletedAt.Add(time.Second))))
	if w.Code != http.StatusOK {
		t.Fatalf("resurrection returned %d, want 200: %s", w.Code, w.Body.String())
	}

	w = request(t, h, http.MethodGet, "/api/score/deleted", nil)
	body = decodeResponse[deletedScoresResponse](t, w)
	if len(body.ScoreIDs) != 0 || len(body.DeletedScores) != 0 {
		t.Errorf("after the resurrection the deleted list is %v / %v, want empty", body.ScoreIDs, body.DeletedScores)
	}
}

func TestDeletedPracticeRecordsReportDeletedAt(t *testing.T) {
	ctx, q, h := setupHandler(t)

	w := request(t, h, http.MethodPost, "/api/practice/record/record-1", newPracticeRecordRequest(time.Unix(1000, 0)))
	if w.Code != http.StatusOK {
		t.Fatalf("upsert returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/practice/record/record-1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	w = request(t, h, http.MethodDelete, "/api/practice/record/record-1", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("second delete returned %d, want 404", w.Code)
	}
	marker, err := q.FindDeletedPracticeRecordMarker(ctx, database.FindDeletedPracticeRecordMarkerParams{User: testUser, RecordID: "record-1"})
	if err != nil {
		t.Fatalf("find deleted practice record marker: %s", err)
	}

	w = request(t, h, http.MethodGet, "/api/practice/record/deleted", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("deleted practice records returned %d: %s", w.Code, w.Body.String())
	}
	body := decodeResponse[deletedResponse](t, w)
	if len(body.Deleted) != 1 || body.Deleted[0].ID != "record-1" {
		t.Fatalf("deleted practice records are %v, want one entry for record-1", body.Deleted)
	}
	if !body.Deleted[0].DeletedAt.Equal(marker.DeletedAt) {
		t.Errorf("deleted[0].deletedAt is %s, want %s", body.Deleted[0].DeletedAt, marker.DeletedAt)
	}
}
