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

type exerciseResponse struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	CategoryID *string         `json:"categoryId"`
	TagIDs     []string        `json:"tagIds"`
	ScoreIDs   []string        `json:"scoreIds"`
	Metadata   json.RawMessage `json:"metadata"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

func newExerciseResponse(exercise database.Exercise, tagIDs, scoreIDs []string) exerciseResponse {
	if tagIDs == nil {
		tagIDs = make([]string, 0)
	}
	if scoreIDs == nil {
		scoreIDs = make([]string, 0)
	}
	return exerciseResponse{
		ID:         exercise.ID,
		Name:       exercise.Name,
		CategoryID: nullString(exercise.CategoryID),
		TagIDs:     tagIDs,
		ScoreIDs:   scoreIDs,
		Metadata:   exercise.MetadataJson,
		UpdatedAt:  exercise.UpdatedAt,
	}
}

// GET /api/practice/exercise?changedAfter=<time>
func (h *Handler) handleGetExercises(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)

	changedAfter, ok := parseTime(w, r.URL.Query().Get("changedAfter"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindExercisesChangedAfter(r.Context(), database.FindExercisesChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find exercises changed after time: %w", err))
		return
	}

	tagRows, err := h.Queries.FindExerciseTagIDsChangedAfter(r.Context(), database.FindExerciseTagIDsChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find tag ids of exercises changed after time: %w", err))
		return
	}
	tagIDs := make(map[string][]string, len(rows))
	for _, row := range tagRows {
		tagIDs[row.ExerciseID] = append(tagIDs[row.ExerciseID], row.TagID)
	}

	scoreRows, err := h.Queries.FindExerciseScoreIDsChangedAfter(r.Context(), database.FindExerciseScoreIDsChangedAfterParams{
		User:    user,
		Changed: changedAfter,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find score ids of exercises changed after time: %w", err))
		return
	}
	scoreIDs := make(map[string][]string, len(rows))
	for _, row := range scoreRows {
		scoreIDs[row.ExerciseID] = append(scoreIDs[row.ExerciseID], row.ScoreID)
	}

	exercises := make([]exerciseResponse, 0, len(rows))
	for _, row := range rows {
		exercises = append(exercises, newExerciseResponse(row, tagIDs[row.ID], scoreIDs[row.ID]))
	}

	type response struct {
		Exercises []exerciseResponse `json:"exercises"`
	}
	respond(w, response{
		Exercises: exercises,
	}, http.StatusOK)
}

// GET /api/practice/exercise/deleted?since=<time>
func (h *Handler) handleGetDeletedExercises(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	since, ok := parseTime(w, r.URL.Query().Get("since"), time.Unix(0, 0))
	if !ok {
		return
	}

	rows, err := h.Queries.FindDeletedExercisesSince(r.Context(), database.FindDeletedExercisesSinceParams{
		User:      user,
		DeletedAt: since,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find deleted exercises since time: %w", err))
		return
	}

	deleted := make([]deletedItem, 0, len(rows))
	for _, row := range rows {
		deleted = append(deleted, deletedItem{ID: row.ExerciseID, DeletedAt: row.DeletedAt})
	}

	type response struct {
		Deleted []deletedItem `json:"deleted"`
	}
	respond(w, response{
		Deleted: deleted,
	}, http.StatusOK)
}

// GET /api/practice/exercise/:id
func (h *Handler) handleGetExercise(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	exercise, err := h.Queries.FindExercise(r.Context(), database.FindExerciseParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("find exercise: %w", err))
		return
	}

	tagIDs, err := h.Queries.GetExerciseTagIDs(r.Context(), database.GetExerciseTagIDsParams{
		User:       user,
		ExerciseID: id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("get exercise tag ids: %w", err))
		return
	}

	scoreIDs, err := h.Queries.GetExerciseScoreIDs(r.Context(), database.GetExerciseScoreIDsParams{
		User:       user,
		ExerciseID: id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("get exercise score ids: %w", err))
		return
	}

	respond(w, newExerciseResponse(exercise, tagIDs, scoreIDs), http.StatusOK)
}

// POST /api/practice/exercise/:id
func (h *Handler) handleUpdateExercise(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")
	if id == "" {
		respondBadRequest(w)
		return
	}

	type request struct {
		Name       string          `json:"name"`
		CategoryID *string         `json:"categoryId"`
		TagIDs     []string        `json:"tagIds"`
		ScoreIDs   []string        `json:"scoreIds"`
		Metadata   json.RawMessage `json:"metadata"`
		UpdatedAt  time.Time       `json:"updatedAt"`
		WrittenAt  time.Time       `json:"writtenAt"`
	}
	params, ok := decodeBody[request](w, r)
	if !ok {
		return
	}

	if params.Name == "" || params.UpdatedAt.IsZero() || params.Metadata == nil {
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

	exercise, err := q.FindExercise(r.Context(), database.FindExerciseParams{
		User: user,
		ID:   id,
	})
	if err == nil {
		if staleUpdate(w, "updatedAt", exercise.UpdatedAt, params.UpdatedAt) {
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		respondErr(w, fmt.Errorf("find exercise: %w", err))
		return
	}

	ok = resolveTombstone(w, params.WrittenAt, tombstone{
		find: func() (time.Time, error) {
			marker, err := q.FindDeletedExerciseMarker(r.Context(), database.FindDeletedExerciseMarkerParams{
				User:       user,
				ExerciseID: id,
			})
			return marker.DeletedAt, err
		},
		remove: func() error {
			return q.DeleteDeletedExerciseMarker(r.Context(), database.DeleteDeletedExerciseMarkerParams{
				User:       user,
				ExerciseID: id,
			})
		},
	})
	if !ok {
		return
	}

	err = q.UpsertExercise(r.Context(), database.UpsertExerciseParams{
		ID:           id,
		User:         user,
		UpdatedAt:    params.UpdatedAt,
		Name:         params.Name,
		CategoryID:   toNullString(params.CategoryID),
		MetadataJson: params.Metadata,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("upsert exercise: %w", err))
		return
	}

	err = q.UnassignAllExerciseTags(r.Context(), database.UnassignAllExerciseTagsParams{
		User:       user,
		ExerciseID: id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("unassign exercise tags: %w", err))
		return
	}

	for _, tagID := range params.TagIDs {
		_, err = q.FindTag(r.Context(), database.FindTagParams{
			User: user,
			ID:   tagID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				respondBadRequest(w)
			} else {
				respondErr(w, fmt.Errorf("find tag %s: %w", tagID, err))
			}
			return
		}

		err = q.AssignExerciseTag(r.Context(), database.AssignExerciseTagParams{
			User:       user,
			ExerciseID: id,
			TagID:      tagID,
		})
		if err != nil {
			respondErr(w, fmt.Errorf("assign tag %s: %w", tagID, err))
			return
		}
	}

	err = q.RemoveAllExerciseScores(r.Context(), database.RemoveAllExerciseScoresParams{
		User:       user,
		ExerciseID: id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("remove exercise scores: %w", err))
		return
	}

	for i, scoreID := range params.ScoreIDs {
		err = q.AddExerciseScore(r.Context(), database.AddExerciseScoreParams{
			User:       user,
			ExerciseID: id,
			Position:   int64(i),
			ScoreID:    scoreID,
		})
		if err != nil {
			respondErr(w, fmt.Errorf("add exercise score %d: %w", i, err))
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}

// DELETE /api/practice/exercise/:id
func (h *Handler) handleDeleteExercise(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	id := chi.URLParam(r, "id")

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	q := h.Queries.WithTx(tx)
	result, err := q.DeleteExercise(r.Context(), database.DeleteExerciseParams{
		User: user,
		ID:   id,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("delete exercise: %w", err))
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

	err = q.CreateDeletedExerciseMarker(r.Context(), database.CreateDeletedExerciseMarkerParams{
		ExerciseID: id,
		User:       user,
	})
	if err != nil {
		respondErr(w, fmt.Errorf("create deleted exercise marker: %w", err))
		return
	}

	err = tx.Commit()
	if err != nil {
		respondInternalServerError(w, fmt.Errorf("commit tx: %w", err))
		return
	}

	respondOK(w)
}
