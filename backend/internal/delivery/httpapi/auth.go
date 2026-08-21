package httpapi

import (
	"context"
	"net/http"
)

type authenticator interface {
	Authenticate(context.Context, string, string) (int64, bool, error)
}

type userIDContextKey struct{}

func basicAuth(auth authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				writeUnauthorized(w)
				return
			}
			userID, authenticated, err := auth.Authenticate(r.Context(), username, password)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication unavailable"})
				return
			}
			if !authenticated || userID <= 0 {
				writeUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDContextKey{}, userID)))
		})
	}
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(int64)
	return userID, ok && userID > 0
}

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authenticated user is missing"})
	}
	return userID, ok
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="rainbet", charset="UTF-8"`)
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
