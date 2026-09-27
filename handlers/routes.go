package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/juho05/sheetopia-sync"
)

func (h *Handler) registerRoutes() {
	h.router.Use(middleware.Recoverer)
	h.router.Use(middleware.StripSlashes)

	h.router.Get("/api/info", h.handleInfo)
	h.router.Post("/api/login", h.handleLogin)

	h.router.Group(func(r chi.Router) {
		r.Use(h.auth)

		r.Post("/api/logout", h.handleLogout)
		r.Get("/api/user", h.handleUser)

		r.Get("/api/score", h.handleGetScores)
		r.Get("/api/score/deleted", h.handleGetDeletedScores)
		r.Get("/api/score/{id}", h.handleGetScore)
		r.Post("/api/score/{id}", h.handleUpdateScore)
		r.Delete("/api/score/{id}", h.handleDeleteScore)

		r.Get("/api/score/{id}/file", h.handleGetScoreFile)
		r.Post("/api/score/{id}/file", h.handleUpdateScoreFile)

		r.Get("/api/tag", h.handleGetTags)
		r.Get("/api/tag/deleted", h.handleGetDeletedTags)
		r.Get("/api/tag/{id}", h.handleGetTag)
		r.Post("/api/tag/{id}", h.handleUpdateTag)
		r.Delete("/api/tag/{id}", h.handleDeleteTag)

		r.Get("/api/setlist", h.handleGetSetlists)
		r.Get("/api/setlist/deleted", h.handleGetDeletedSetlists)
		r.Get("/api/setlist/{id}", h.handleGetSetlist)
		r.Post("/api/setlist/{id}", h.handleUpdateSetlist)
		r.Delete("/api/setlist/{id}", h.handleDeleteSetlist)

		r.Get("/api/practice/category", h.handleGetExerciseCategories)
		r.Get("/api/practice/category/deleted", h.handleGetDeletedExerciseCategories)
		r.Get("/api/practice/category/{id}", h.handleGetExerciseCategory)
		r.Post("/api/practice/category/{id}", h.handleUpdateExerciseCategory)
		r.Delete("/api/practice/category/{id}", h.handleDeleteExerciseCategory)

		r.Get("/api/practice/exercise", h.handleGetExercises)
		r.Get("/api/practice/exercise/deleted", h.handleGetDeletedExercises)
		r.Get("/api/practice/exercise/{id}", h.handleGetExercise)
		r.Post("/api/practice/exercise/{id}", h.handleUpdateExercise)
		r.Delete("/api/practice/exercise/{id}", h.handleDeleteExercise)

		r.Get("/api/practice/routine", h.handleGetPracticeRoutines)
		r.Get("/api/practice/routine/deleted", h.handleGetDeletedPracticeRoutines)
		r.Get("/api/practice/routine/{id}", h.handleGetPracticeRoutine)
		r.Post("/api/practice/routine/{id}", h.handleUpdatePracticeRoutine)
		r.Delete("/api/practice/routine/{id}", h.handleDeletePracticeRoutine)

		r.Get("/api/practice/record", h.handleGetPracticeRecords)
		r.Get("/api/practice/record/deleted", h.handleGetDeletedPracticeRecords)
		r.Get("/api/practice/record/{id}", h.handleGetPracticeRecord)
		r.Post("/api/practice/record/{id}", h.handleUpdatePracticeRecord)
		r.Delete("/api/practice/record/{id}", h.handleDeletePracticeRecord)
	})
}

// GET /api/info
func (h *Handler) handleInfo(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Server        string    `json:"server"`
		Time          time.Time `json:"time"`
		ServerVersion string    `json:"serverVersion"`
		APIVersion    string    `json:"apiVersion"`
	}
	respond(w, response{
		Server:        "sheetopia-sync",
		Time:          time.Now(),
		ServerVersion: sheetopia.Version,
		APIVersion:    sheetopia.APIVersion,
	}, http.StatusOK)
}
