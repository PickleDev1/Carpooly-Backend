package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	jwt.RegisteredClaims
	EmailAddress string `json:"email"`
}

type authContextKey int

const (
	userIDKey authContextKey = iota
	emailKey
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
	// Get JWKS URL from environment variable
	jwksURL := os.Getenv("CLERK_JWKS_URL")
	if jwksURL == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CLERK_JWKS_URL environment variable not set\"}")
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Server configuration error", http.StatusInternalServerError)
			})
		}
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Initializing auth middleware\",\"jwks_url\":\"%s\"}", jwksURL)

	// Create the JWKS from the URL
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
		RefreshInterval: time.Hour,
	})
	if err != nil {
		log.Printf("{\"severity\":\"FATAL\",\"message\":\"Failed to create JWKS from URL\",\"error\":\"%v\"}", err)
		panic(fmt.Sprintf("Failed to create JWKS from URL: %v", err))
	}

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

			// Log the entire request for debugging
			for name, values := range r.Header {
				log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Header\",\"name\":\"%s\",\"value\":\"%s\"}",
					name, values[0])
			}

			// Get the Authorization header
			authHeader := r.Header.Get("Authorization")
			//log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Auth header received\",\"header\":\"%s\"}", authHeader)

			if authHeader == "" {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"No Authorization header provided\"}")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check for Authorization header
			if !strings.HasPrefix(authHeader, "Bearer ") {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid auth header format\"}")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Auth header received\",\"header\":\"%s\"}", authHeader)

			// Extract token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse and validate the token
			token, err := jwt.Parse(tokenString, jwks.Keyfunc)
			if err != nil || !token.Valid {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"Token validation failed\",\"error\":\"%v\",\"token_valid\":%v}",
					err, token != nil && token.Valid)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Token validated successfully\"}")

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid claims format\"}")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get user info from claims
			userID, ok := claims["sub"].(string)
			if !ok {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"No user ID in claims\"}")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Claims extracted\",\"user_id\":\"%s\"}", userID)

			// Add user info to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, userIDKey, userID)
			if email, ok := claims["email"].(string); ok {
				ctx = context.WithValue(ctx, emailKey, email)
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
