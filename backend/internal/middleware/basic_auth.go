package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"rainbet/internal/response"
)

func BasicAuth(username, password string) func(http.Handler) http.Handler {
	usernameHash := sha256.Sum256([]byte(username))
	passwordHash := sha256.Sum256([]byte(password))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedUsername, providedPassword, ok := r.BasicAuth()
			providedUsernameHash := sha256.Sum256([]byte(providedUsername))
			providedPasswordHash := sha256.Sum256([]byte(providedPassword))

			credentialsMatch := subtle.ConstantTimeCompare(providedUsernameHash[:], usernameHash[:])&
				subtle.ConstantTimeCompare(providedPasswordHash[:], passwordHash[:]) == 1
			if !ok || !credentialsMatch {
				w.Header().Set("WWW-Authenticate", `Basic realm="rainbet", charset="UTF-8"`)
				response.JSON(w, http.StatusUnauthorized, map[string]string{
					"error": "unauthorized",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
