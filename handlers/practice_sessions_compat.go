package handlers

import "net/http"

// Stubs for apps that still sync practice sessions, so their sync of everything else keeps
// succeeding until they are updated.
// TODO remove once all testers run a build with new session storage

// GET /api/practice/session
func (h *Handler) handleGetPracticeSessionsCompat(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Sessions []struct{} `json:"sessions"`
	}
	respond(w, response{Sessions: []struct{}{}}, http.StatusOK)
}

// GET /api/practice/session/deleted
func (h *Handler) handleGetDeletedPracticeSessionsCompat(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Deleted []deletedItem `json:"deleted"`
	}
	respond(w, response{Deleted: []deletedItem{}}, http.StatusOK)
}

// POST /api/practice/session/:id
func (h *Handler) handleUpdatePracticeSessionCompat(w http.ResponseWriter, r *http.Request) {
	respondOK(w)
}

// GET and DELETE /api/practice/session/:id
func (h *Handler) handlePracticeSessionNotFoundCompat(w http.ResponseWriter, r *http.Request) {
	respondNotFound(w)
}
