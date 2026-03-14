package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"mockBackend/internal/api"
)

func main() {
	fmt.Println("--- mock backend started ---")
	
	// TODO: add url to .env
	dsn := "postgres://admin:password@localhost:5432/mock_db?sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("error while connecting to DB: %s\n", err)
	}
	defer db.Close()
	fmt.Println("db connected")

	r := chi.NewRouter()
	
	r.Use(api.CorsHandler.Handler) 	// CORS
	r.Use(middleware.Logger)		// LOGGER
	r.Use(middleware.Recoverer)		// PANIC RECOVER

	r.Post("/api/register", api.RegisterHandler(db))

	r.Group(func(r chi.Router) {
		r.Use(api.AuthMiddleware)
		r.Post("/api/subjects", api.CreateSubjectHandler(db))
		r.Get("/api/subjects", api.GetSubjectsHandler(db))
	})

	port := ":8080"
	fmt.Printf("server is running on %s", port)
	http.ListenAndServe(port, r)
}
