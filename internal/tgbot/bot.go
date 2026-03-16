package tgbot

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"strconv"
	"strings"
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

		var startProvided, endProvided bool
		var startTs, endTs int64

		text := c.Message().Text
		parts := strings.Fields(text)

		for _, p := range parts[1:] {
			if strings.HasPrefix(p, "start_date=") {
				if v, err := strconv.ParseInt(strings.TrimPrefix(p, "start_date="), 10, 64); err == nil {
					startProvided = true
					startTs = v
				}
			} else if strings.HasPrefix(p, "end_date=") {
				if v, err := strconv.ParseInt(strings.TrimPrefix(p, "end_date="), 10, 64); err == nil {
					endProvided = true
					endTs = v
				}
			}
		}

		logs, err := storage.GetAllNavigationLogs(b.db)
		if err != nil {
			return c.Send("Ошибка получения логов.")
		}

		filtered := make([]storage.NavigationLog, 0, len(logs))
		for _, l := range logs {
			ts := l.CreatedAt.Unix()

			if startProvided && ts < startTs {
				continue
			}
			if endProvided && ts > endTs {
				continue
			}

			filtered = append(filtered, l)
		}

		buf := new(bytes.Buffer)
		writer := csv.NewWriter(buf)
		err = writer.Write([]string{"Timestamp", "UserID", "TabName"})
		if err != nil {
			return err
		}

		for _, l := range filtered {
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
