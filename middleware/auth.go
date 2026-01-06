package middleware

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
)

type authContextKey int

const (
	userIDKey authContextKey = iota
	emailKey
	timezoneKey
)

// Add this new type to capture the response
type responseWriter struct {
	http.ResponseWriter
	body *strings.Builder
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Write to both the original ResponseWriter and our buffer
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func AuthMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Request received\",\"path\":\"%s\",\"method\":\"%s\"}",
				r.URL.Path, r.Method)

			// Create a response wrapper if this is the /api/invites endpoint
			var rw *responseWriter
			if r.URL.Path == "/api/invites" {
				rw = &responseWriter{
					ResponseWriter: w,
					body:           &strings.Builder{},
				}
				w = rw
			}

			// Allow OPTIONS requests to pass through
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			log.Printf("{\"severity\":\"INFO\",\"message\":\"Starting AuthMiddleware\",\"path\":\"%s\"}", r.URL.Path)

			// First, check if Clerk's claims are already in the context (from Clerk's middleware)
			ctx := r.Context()
			clerkClaims, hasClerkClaims := clerk.SessionClaimsFromContext(ctx)
			
			// Log authorization header for debugging (without the actual token)
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				if strings.HasPrefix(authHeader, "Bearer ") {
					previewLen := 20
					if len(authHeader) < previewLen {
						previewLen = len(authHeader)
					}
					tokenPreview := authHeader[:previewLen] + "..."
					log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Authorization header present\",\"preview\":\"%s\",\"length\":%d}", tokenPreview, len(authHeader))
				} else {
					log.Printf("{\"severity\":\"WARNING\",\"message\":\"Authorization header missing Bearer prefix\"}")
				}
			} else {
				log.Printf("{\"severity\":\"WARNING\",\"message\":\"No Authorization header found\"}")
			}
			
			if !hasClerkClaims {
				// Log detailed error information
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"No Clerk claims in context - Clerk middleware may have failed\",\"path\":\"%s\",\"method\":\"%s\",\"has_auth_header\":%t}", 
					r.URL.Path, r.Method, authHeader != "")
				
				// Check if CLERK_SECRET_KEY is set (without logging the actual value)
				clerkKeySet := os.Getenv("CLERK_SECRET_KEY") != ""
				log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CLERK_SECRET_KEY is set\",\"is_set\":%t}", clerkKeySet)
				
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get user ID from Clerk's claims
			userID := clerkClaims.Subject
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Clerk claims found\",\"user_id\":\"%s\"}", userID)
			ctx = context.WithValue(ctx, userIDKey, userID)

			// Extract timezone from header
			timezone := r.Header.Get("X-User-Timezone")
			if timezone != "" {
				ctx = context.WithValue(ctx, timezoneKey, timezone)
				log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Timezone extracted from header\",\"timezone\":\"%s\"}", timezone)
			} else {
				// Default to UTC if no timezone provided
				ctx = context.WithValue(ctx, timezoneKey, "UTC")
				log.Printf("{\"severity\":\"DEBUG\",\"message\":\"No timezone header, defaulting to UTC\"}")
			}

			// Before calling next handler
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Proceeding to handler\",\"path\":\"%s\"}", r.URL.Path)
			next.ServeHTTP(w, r.WithContext(ctx))
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Handler completed\",\"path\":\"%s\"}", r.URL.Path)

			// Log the response if this was the /api/invites endpoint
			if rw != nil {
				log.Printf("{\"severity\":\"INFO\",\"message\":\"Invite response\",\"response\":%s}", rw.body.String())
			}
		})
	}
}

// Get userID from context
func GetClerkIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

// Add a helper function to get email
func GetEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(emailKey).(string)
	return email, ok
}

// Add a helper function to get timezone
func GetTimezoneFromContext(ctx context.Context) (string, bool) {
	timezone, ok := ctx.Value(timezoneKey).(string)
	return timezone, ok
}
