package middleware

import (
	"context"
	"net/http"

	"rainbet/internal/response"
)

type Authenticator interface {
	Authenticate(ctx context.Context, username, password string) (bool, error)
}

func BasicAuth(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedUsername, providedPassword, ok := r.BasicAuth()
			if !ok {
				writeUnauthorized(w)
				return
			}

			authenticated, err := authenticator.Authenticate(r.Context(), providedUsername, providedPassword)
			if err != nil {
				response.JSON(w, http.StatusInternalServerError, map[string]string{
					"error": "authentication unavailable",
				})
				return
			}
			if !authenticated {
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="rainbet", charset="UTF-8"`)
	response.JSON(w, http.StatusUnauthorized, map[string]string{
		"error": "unauthorized",
	})
}
