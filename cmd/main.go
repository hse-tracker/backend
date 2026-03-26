package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/hse-tracker/backend/internal/api"
	"github.com/hse-tracker/backend/internal/config"
	"github.com/hse-tracker/backend/internal/parser"
	"github.com/hse-tracker/backend/internal/tgbot"
	database "github.com/hse-tracker/backend/internal/db"
)

func main() {
	fmt.Println("- - - backend started - - -")

	// load vars from config
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// create DB
	db, err := sqlx.Connect("postgres", cfg.DBPath)
	if err != nil {
		log.Fatalf("error while connecting to DB: %s\n", err)
	}

	// run migrations
	if err := database.RunMigrations(db); err != nil {
        log.Fatalf("db error: %v", err)
    }

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("error while closing DB: %s\n", err)
		}
	}(db)
	fmt.Println("db connected")

	// create chi router
	r := chi.NewRouter()

	corsMiddleware := api.NewCorsHandler(cfg)

	r.Use(corsMiddleware.Handler) // CORS func
	r.Use(middleware.Logger)      // LOGGER
	r.Use(middleware.Recoverer)   // PANIC RECOVER

	r.Post("/api/register", api.RegisterHandler(db, cfg.JWTSecret, cfg.TelegramToken))

	r.Group(func(r chi.Router) {
		r.Use(api.AuthMiddleware(cfg.JWTSecret))
		r.Post("/api/subjects", api.CreateSubjectHandler(db, cfg))
		r.Get("/api/subjects", api.GetSubjectsHandler(db))
		r.Get("/api/subjects/{id}", api.GetSubjectHandler(db))
		r.Delete("/api/subjects/{id}", api.DeleteSubjectHandler(db))
		r.Post("/api/logs/navigation", api.LogNavigationHandler(db))
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

	timeGap := 1 * time.Minute
	parser.StartSyncWorker(db, cfg.GoogleCredsPath, gradesBot, timeGap)

	err = http.ListenAndServe(port, r)
	if err != nil {
		return
	}
}
