package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/juho05/sheetopia-sync/database"
)

func (h *Handler) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			respondUnauthorized(w)
			return
		}
		authKey := strings.TrimPrefix(authHeader, "Bearer ")

		algos := []database.HashAlg{
			database.HashAlg_SHA256,
			database.HashAlgPBKDF2_SHA256_10000,

			// The algorithm this points to should already be at the top of the list above.
			// Try it again at the end to catch concurrent requests where one rewrites the auth key before the other
			// one is verified.
			database.HashAlg_DEFAULT,
		}

		var user string
		var successAlg database.HashAlg
		var err error
		for _, alg := range algos {
			user, err = h.Queries.VerifyAuthKey(r.Context(), database.VerifyAuthKeyParams{
				KeyHash: database.HashAuthKey(authKey, alg),
				HashAlg: string(alg),
			})
			if err == nil {
				successAlg = alg
				break
			}
			if !errors.Is(err, sql.ErrNoRows) {
				respondInternalServerError(w, err)
				return
			}
		}

		if user == "" || successAlg == "" {
			respondUnauthorized(w)
			return
		}

		if successAlg != database.HashAlg_DEFAULT {
			err = h.Queries.MigrateAuthKey(r.Context(), database.MigrateAuthKeyParams{
				NewKeyHash: database.HashAuthKey(authKey, database.HashAlg_DEFAULT),
				NewHashAlg: string(database.HashAlg_DEFAULT),
				OldKeyHash: database.HashAuthKey(authKey, successAlg),
				OldHashAlg: string(successAlg),
			})
			if err != nil {
				respondInternalServerError(w, err)
				return
			}
		}

		r = r.WithContext(context.WithValue(r.Context(), "user", user))
		next.ServeHTTP(w, r)
	})
}
