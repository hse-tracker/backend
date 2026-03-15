package tgbot

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"time"

	"github.com/hse-tracker/backend/internal/storage"
	"github.com/jmoiron/sqlx"

	"gopkg.in/telebot.v3"
)

type Bot struct {
	token string
	db    *sqlx.DB
	api   *telebot.Bot
}

func New(token string, db *sqlx.DB) *Bot {
	return &Bot{
		token: token,
		db:    db,
	}
}

func (b *Bot) Start(adminIDs []int64) error {
	pref := telebot.Settings{
		Token:  b.token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		return fmt.Errorf("failed to create bot instance: %w", err)
	}
	b.api = bot

	bot.Handle("/start", func(c telebot.Context) error {
		user := c.Sender()

		fullName := user.FirstName
		if user.LastName != "" {
			fullName += " " + user.LastName
		}
		if fullName == "" {
			fullName = user.Username
		}

		log.Printf("user %d (%s) writing to bot", user.ID, fullName)
		return c.Send(fmt.Sprintf("Привет, %s! Я бот для отслеживания оценок в Вышке. Нажми кнопку, чтобы перейти в мини-приложение", fullName))
	})

	bot.Handle("/logs", func(c telebot.Context) error {
		isAdmin := false
		for _, id := range adminIDs {
			if c.Sender().ID == id {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			return c.Send("У вас нет прав администратора.")
		}

		logs, err := storage.GetAllNavigationLogs(b.db)
		if err != nil {
			return c.Send("Ошибка получения логов.")
		}

		buf := new(bytes.Buffer)
		writer := csv.NewWriter(buf)
		err = writer.Write([]string{"Timestamp", "UserID", "TabName"})
		if err != nil {
			return err
		}

		for _, l := range logs {
			err := writer.Write([]string{
				l.CreatedAt.Format(time.RFC3339),
				fmt.Sprintf("%d", l.UserID),
				l.TabName,
			})
			if err != nil {
				return err
			}
		}
		writer.Flush()

		doc := &telebot.Document{
			File:     telebot.FromReader(buf),
			FileName: "navigation_logs.csv",
		}
		return c.Send(doc)
	})

	log.Println("Telegram Bot started!")
	bot.Start()
	return nil
}

func (b *Bot) SendMessage(userID int64, text string) error {
	if b.api == nil {
		return fmt.Errorf("bot is not initialized")
	}

	recipient := &telebot.Chat{ID: userID}

	_, err := b.api.Send(recipient, text)
	return err
}
