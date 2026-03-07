package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// CORS returns a handler that adds CORS headers.
func CORS(origins []string) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	return c.Handler
}
