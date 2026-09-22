package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

type deletedItem struct {
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deletedAt"`
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func toNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func getUser(r *http.Request) string {
	user := r.Context().Value("user")
	if user == nil {
		panic(errors.New("user not found in context"))
	}
	userStr := user.(string)
	if len(userStr) == 0 {
		panic(errors.New("user is empty"))
	}
	return userStr
}

func respond[T any](w http.ResponseWriter, v T, status int) {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = encoder.Encode(v)
}

func respondOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

func respondConflict(w http.ResponseWriter) {
	respondStatus(w, http.StatusConflict)
}

func respondUnauthorized(w http.ResponseWriter) {
	respondStatus(w, http.StatusUnauthorized)
}

func respondErr(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		respondNotFound(w)
		return
	}
	respondInternalServerError(w, err)
}

func respondInternalServerError(w http.ResponseWriter, err error) {
	log.Printf("Internal Server Error: %s", err)
	respondStatus(w, http.StatusInternalServerError)
}

func respondNotFound(w http.ResponseWriter) {
	respondStatus(w, http.StatusNotFound)
}

func respondBadRequest(w http.ResponseWriter) {
	respondStatus(w, http.StatusBadRequest)
}

func respondStatus(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(http.StatusText(status)))
}

func parseTime(w http.ResponseWriter, str string, def time.Time) (time.Time, bool) {
	if str != "" {
		var err error
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			respondBadRequest(w)
			return time.Time{}, false
		}
		return t, true
	}
	return def, true
}

func decodeBody[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var obj T
	err := json.NewDecoder(r.Body).Decode(&obj)
	_ = r.Body.Close()
	if err != nil {
		respondBadRequest(w)
		return obj, false
	}
	return obj, true
}

// staleUpdate reports whether the stored row is at least as new as the incoming write. It responds
// with the stored timestamp under key in that case so that the client can settle its local write.
func staleUpdate(w http.ResponseWriter, key string, stored, incoming time.Time) bool {
	if stored.Before(incoming) {
		return false
	}
	respond(w, map[string]time.Time{
		key: stored,
	}, http.StatusConflict)
	return true
}

type tombstone struct {
	find   func() (time.Time, error)
	remove func() error
}

// resolveTombstone reports whether a write may proceed for a row that may have been deleted before.
// Only a write that an import restored after the deletion wins. Removing the marker keeps other
// devices from being handed a deletion for a live row.
func resolveTombstone(w http.ResponseWriter, writtenAt time.Time, t tombstone) bool {
	deletedAt, err := t.find()
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	if err != nil {
		respondErr(w, fmt.Errorf("check if already deleted: %w", err))
		return false
	}

	if writtenAt.IsZero() || !writtenAt.After(deletedAt) {
		type response struct {
			DeletedAt time.Time `json:"deletedAt"`
		}
		respond(w, response{
			DeletedAt: deletedAt,
		}, http.StatusConflict)
		return false
	}

	err = t.remove()
	if err != nil {
		respondErr(w, fmt.Errorf("delete deleted marker: %w", err))
		return false
	}
	return true
}
