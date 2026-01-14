package main

import (
	"fmt"
	"log"

	"github.com/AntiSlang/tracker/docs"
	"github.com/AntiSlang/tracker/internal/adminserver"
	"github.com/AntiSlang/tracker/internal/config"
	"github.com/AntiSlang/tracker/internal/storage"
	"github.com/AntiSlang/tracker/internal/tgbot"
	"github.com/AntiSlang/tracker/internal/tools"
)

// @title HSE Tracker API
// @version 1.0
// @description API мониторинга оценок hse-tracker
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	log.Println("HSE Tracker starting...")

	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := storage.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer db.Close()
	log.Println("Database connection established.")

	gradesBot := tgbot.New(cfg.TelegramToken, db)

	adminApiServer := adminserver.New(cfg, db)

	go func() {
		log.Println("Starting Telegram Bot...")
		if err := gradesBot.Start(); err != nil {
			log.Fatalf("Failed to start Telegram Bot: %v", err)
		}
	}()

	go func() {
		docs.SwaggerInfo.Host = fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
		log.Printf("Swagger UI configured for host: %s", docs.SwaggerInfo.Host)

		if err := adminApiServer.Run(); err != nil {
			log.Fatalf("Failed to start Admin API server: %v", err)
		}
	}()

	tools.WaitForShutdownSignal("HSE Tracker")
}
