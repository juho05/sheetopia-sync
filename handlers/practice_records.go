package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/juho05/sheetopia-sync/database"
)

type practiceRecordResponse struct {
	ID             string          `json:"id"`
	ExerciseID     string          `json:"exerciseId"`
	RoutineID      *string         `json:"routineId"`
	RoutineEntryID *string         `json:"routineEntryId"`
	Metadata       json.RawMessage `json:"metadata"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

func newPracticeRecordResponse(record database.PracticeRecord) practiceRecordResponse {
	return practiceRecordResponse{
		ID:             record.ID,
		ExerciseID:     record.ExerciseID,
		RoutineID:      nullString(record.RoutineID),
		RoutineEntryID: nullString(record.RoutineEntryID),
		Metadata:       record.MetadataJson,
		UpdatedAt:      record.UpdatedAt,
	}
}

// GET /api/practice/record?changedAfter=<time>
func (h *Handler) handleGetPracticeRecords(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	changedAfter, ok := parseTime(w, r.URL.Query().Get("changedAfter"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindPracticeRecordsChangedAfter(r.Context(), database.FindPracticeRecordsChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find practice records changed after time: %w", err))
		return
	}

	records := make([]practiceRecordResponse, 0, len(rows))
	for _, row := range rows {
		records = append(records, newPracticeRecordResponse(row))
	}

	type response struct {
		Records []practiceRecordResponse `json:"records"`
	}
	respond(w, response{
		Records: records,
	}, http.StatusOK)
}

// GET /api/practice/record/deleted?since=<time>
func (h *Handler) handleGetDeletedPracticeRecords(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	since, ok := parseTime(w, r.URL.Query().Get("since"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindDeletedPracticeRecordsSince(r.Context(), database.FindDeletedPracticeRecordsSinceParams{
		User:      user,
		DeletedAt: since,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find deleted practice records since time: %w", err))
		return
	}

	deleted := make([]deletedItem, 0, len(rows))
	for _, row := range rows {
		deleted = append(deleted, deletedItem{ID: row.RecordID, DeletedAt: row.DeletedAt})
	}

	type response struct {
		Deleted []deletedItem `json:"deleted"`
	}
	respond(w, response{
		Deleted: deleted,
	}, http.StatusOK)
}

// GET /api/practice/record/:id
func (h *Handler) handleGetPracticeRecord(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	record, err := h.Queries.FindPracticeRecord(r.Context(), database.FindPracticeRecordParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find practice record: %w", err))
		return
	}

	respond(w, newPracticeRecordResponse(record), http.StatusOK)
}

// POST /api/practice/record/:id
func (h *Handler) handleUpdatePracticeRecord(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		respondBadRequest(w)
		return
	}

	type request struct {
		ExerciseID     string          `json:"exerciseId"`
		RoutineID      *string         `json:"routineId"`
		RoutineEntryID *string         `json:"routineEntryId"`
		Metadata       json.RawMessage `json:"metadata"`
		UpdatedAt      time.Time       `json:"updatedAt"`
		WrittenAt      time.Time       `json:"writtenAt"`
	}
	params, ok := decodeBody[request](w, r)
	if !ok {
		return
	}

	if params.ExerciseID == "" || params.UpdatedAt.IsZero() || params.Metadata == nil {
		respondBadRequest(w)
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)

	record, err := q.FindPracticeRecord(r.Context(), database.FindPracticeRecordParams{
		User: user,
		ID:   id,
	})
	if err == nil {
		if staleUpdate(w, "updatedAt", record.UpdatedAt, params.UpdatedAt) {
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("find practice record: %w", err))
		return
	}

	ok = resolveTombstone(w, params.WrittenAt, tombstone{
		find: func() (time.Time, error) {
			marker, err := q.FindDeletedPracticeRecordMarker(r.Context(), database.FindDeletedPracticeRecordMarkerParams{
				User:     user,
				RecordID: id,
			})
			return marker.DeletedAt, err
		},
		remove: func() error {
			return q.DeleteDeletedPracticeRecordMarker(r.Context(), database.DeleteDeletedPracticeRecordMarkerParams{
				User:     user,
				RecordID: id,
			})
		},
	})
	if !ok {
		return
	}

	err = q.UpsertPracticeRecord(r.Context(), database.UpsertPracticeRecordParams{
		ID:             id,
		User:           user,
		UpdatedAt:      params.UpdatedAt,
		ExerciseID:     params.ExerciseID,
		RoutineID:      toNullString(params.RoutineID),
		RoutineEntryID: toNullString(params.RoutineEntryID),
		MetadataJson:   params.Metadata,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("upsert practice record: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}

// DELETE /api/practice/record/:id
func (h *Handler) handleDeletePracticeRecord(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)
	result, err := q.DeletePracticeRecord(r.Context(), database.DeletePracticeRecordParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("delete practice record: %w", err))
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		respondErr(w, fmt.Errorf("affected rows: %w", err))
		return
	}
	if rowsAffected == 0 {
		respondNotFound(w)
		return
	}

	err = q.CreateDeletedPracticeRecordMarker(r.Context(), database.CreateDeletedPracticeRecordMarkerParams{
		RecordID: id,
		User:     user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("create deleted practice record marker: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}
