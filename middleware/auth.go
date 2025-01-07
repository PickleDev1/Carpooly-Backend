package middleware

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
)

type UserClaims struct {
	jwt.StandardClaims
	EmailAddress string `json:"email"`
}

type authContextKey int

const (
	userIDKey authContextKey = iota
	emailKey
)

func AuthMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log request details
			log.Printf("{\"severity\":\"INFO\",\"message\":\"Starting AuthMiddleware\",\"path\":\"%s\",\"method\":\"%s\"}",
				r.URL.Path, r.Method)

			// Log headers
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Request headers\",\"headers\":%q}", r.Header)

			// Check for Authorization header
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract claims from JWT token
			if claims, ok := r.Context().Value("clerk.claims").(*UserClaims); ok {
				userID := claims.Subject // Get user ID from JWT claims
				ctx := context.WithValue(r.Context(), userIDKey, userID)
				ctx = context.WithValue(ctx, emailKey, claims.EmailAddress)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			http.Error(w, "Unauthorized - Invalid claims", http.StatusUnauthorized)
		})
	}
}

// Get userID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

// Add a helper function to get email
func GetEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(emailKey).(string)
	return email, ok
}
