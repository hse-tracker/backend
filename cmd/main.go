package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hse-tracker/backend/internal/config"
	"github.com/hse-tracker/backend/internal/tgbot"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/hse-tracker/backend/internal/api"
)

func main() {
	fmt.Println("--- mock backend started ---")

	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := sqlx.Connect("postgres", cfg.DBPath)
	if err != nil {
		log.Fatalf("error while connecting to DB: %s\n", err)
	}
	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("error while closing DB: %s\n", err)
		}
	}(db)
	fmt.Println("db connected")

	r := chi.NewRouter()

	corsMiddleware := api.NewCorsHandler(cfg)

	r.Use(corsMiddleware.Handler) // CORS func
	r.Use(middleware.Logger)      // LOGGER
	r.Use(middleware.Recoverer)   // PANIC RECOVER

	r.Post("/api/register", api.RegisterHandler(db, cfg.JWTSecret))

	r.Group(func(r chi.Router) {
		r.Use(api.AuthMiddleware(cfg.JWTSecret))
		r.Post("/api/subjects", api.CreateSubjectHandler(db))
		r.Get("/api/subjects", api.GetSubjectsHandler(db))
	})

	port := ":" + cfg.ServerPort
	fmt.Printf("server is running on %s", port)

	gradesBot := tgbot.New(cfg.TelegramToken, db)

	go func() {
		fmt.Println("starting tg bot")
		if err := gradesBot.Start(cfg.AdminIDs); err != nil {
			log.Fatalf("failed to start tg bot: %v", err)
		}
	}()

	err = http.ListenAndServe(port, r)
	if err != nil {
		return
	}
}
