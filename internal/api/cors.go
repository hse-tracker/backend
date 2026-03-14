package api

import (
	"github.com/go-chi/cors"
)

// CORS
var CorsHandler = cors.New(cors.Options{
// TODO: replace "*" with origin domain
	AllowedOrigins: []string{"*"},
	AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH",},
	AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With",},
	ExposedHeaders: []string{"Link", "X-Total-Count",},
	AllowCredentials: true,
	MaxAge: 300, // 5min
})
