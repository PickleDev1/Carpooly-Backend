package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
)

func WebhookAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Starting WebhookAuthMiddleware\",\"path\":\"%s\"}", r.URL.Path)

		// Read and log request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to read request body\",\"error\":\"%v\"}", err)
			http.Error(w, "Failed to read request", http.StatusBadRequest)
			return
		}

		// Log the request body
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Request body\",\"body\":%s}", bodyBytes)

		// Important: Restore the body for later use
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Get webhook secret from environment
		webhookSecret := os.Getenv("WEBHOOK_SECRET")
		if webhookSecret == "" {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"WEBHOOK_SECRET not configured in environment\"}")
			http.Error(w, "Server configuration error", http.StatusInternalServerError)
			return
		}

		// Check webhook secret header
		receivedSecret := r.Header.Get("X-Webhook-Secret")
		log.Printf(`{"severity":"DEBUG","message":"Webhook authentication","receivedSecret":"%s"}`, receivedSecret)

		if receivedSecret == "" {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"No X-Webhook-Secret header provided\"}")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if receivedSecret != webhookSecret {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid webhook secret received\"}")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("{\"severity\":\"INFO\",\"message\":\"Webhook authentication successful\"}")
		next.ServeHTTP(w, r)
	})
}
