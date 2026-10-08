package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/juho05/sheetopia-sync/database"
)

// POST /api/login
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Username string `json:"user"`
		Password string `json:"password"`
	}
	params, ok := decodeBody[request](w, r)
	if !ok {
		return
	}

	user, err := h.Queries.FindUser(r.Context(), params.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondUnauthorized(w)
		} else {
			respondInternalServerError(w, err)
		}
		return
	}

	valid, err := database.VerifyPassword(user.PasswordHash, params.Password)
	if err != nil {
		respondInternalServerError(w, err)
		return
	}
	if !valid {
		respondUnauthorized(w)
		return
	}

	authKey, err := database.GenerateAuthKey()
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	err = h.Queries.CreateAuthKey(r.Context(), database.CreateAuthKeyParams{
		KeyHash: database.HashAuthKey(authKey, database.HashAlg_DEFAULT),
		HashAlg: string(database.HashAlg_DEFAULT),
		User:    user.Name,
	})
	if err != nil {
		respondInternalServerError(w, err)
		return
	}

	type response struct {
		AuthKey string `json:"authKey"`
	}
	respond(w, response{
		AuthKey: authKey,
	}, http.StatusOK)
}

// POST /api/logout
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	authKey := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	// hash alg was already converted to default one by auth middleware at this point
	err := h.Queries.DeleteAuthKey(r.Context(), database.DeleteAuthKeyParams{
		KeyHash: database.HashAuthKey(authKey, database.HashAlg_DEFAULT),
		HashAlg: string(database.HashAlg_DEFAULT),
	})
	if err != nil {
		respondInternalServerError(w, err)
		return
	}
	respondOK(w)
}

// GET /api/user
func (h *Handler) handleUser(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	type response struct {
		User string `json:"user"`
	}
	respond(w, response{
		User: user,
	}, http.StatusOK)
}
