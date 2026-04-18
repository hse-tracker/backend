package tgbot

import (
	"bytes"
	"database/sql"
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
		Token:     b.token,
		Poller:    &telebot.LongPoller{Timeout: 10 * time.Second},
		ParseMode: telebot.ModeMarkdown,
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		return fmt.Errorf("failed to create bot instance: %w", err)
	}
	b.api = bot

	if err := b.ensureReminderTable(); err != nil {
		return fmt.Errorf("failed to ensure reminder table: %w", err)
	}

	go b.reminderWorker()

	bot.Handle("/start", func(c telebot.Context) error {
		_ = b.scheduleMessage(c.Sender().ID)

		user := c.Sender()

		fullName := user.FirstName
		if user.LastName != "" {
			fullName += " " + user.LastName
		}
		if fullName == "" {
			fullName = user.Username
		}

		log.Printf("user %d (%s) writing to bot", user.ID, fullName)
		album := telebot.Album{
			&telebot.Photo{
				File:    telebot.FromDisk("materials/Frame 3.png"),
				Caption: "**Твои оценки теперь в телеграме**\nПрепод вот-вот выставит оценки, и ты весь день проверяешь ведомость.\nЗнакомо? А что, если мы скажем тебе, что больше не нужно этого делать?\nВстречай @hsetrackerbot – бот, который мгновенно пришлет тебе новые оценки в телеграм и подскажет, сколько баллов нужно набрать на экзамене, чтобы закрыть предмет.\nСосредоточься на учебе, а расчетом оценок займется бот @hsetrackerbot!\n\nПодробнее - в карточках",
			},
			&telebot.Photo{File: telebot.FromDisk("materials/Frame 4.png")},
			&telebot.Photo{File: telebot.FromDisk("materials/Frame 5.png")},
			&telebot.Photo{File: telebot.FromDisk("materials/Frame 6.png")},
		}

		return c.SendAlbum(album)
	})

	// На первое обычное текстовое сообщение тоже ставим напоминание.
	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		if strings.HasPrefix(c.Text(), "/") {
			return nil
		}
		return b.scheduleMessage(c.Sender().ID)
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

		log.Printf("[Bot] Admin %d requested navigation logs CSV", c.Sender().ID)

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
			log.Printf(" [Bot] Ошибка получения логов из БД: %v", err)
			return c.Send(fmt.Sprintf("Ошибка получения логов: %v", err))
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

		if err := writer.Error(); err != nil {
			return err
		}

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

func (b *Bot) ensureReminderTable() error {
	_, err := b.db.Exec(`
		CREATE TABLE IF NOT EXISTS tg_reminders (
			user_id BIGINT PRIMARY KEY,
			send_at TIMESTAMPTZ NOT NULL,
			sent_at TIMESTAMPTZ NULL
		)
	`)
	return err
}

func (b *Bot) scheduleMessage(userID int64) error {
	_, err := b.db.Exec(`
		INSERT INTO tg_reminders (user_id, send_at, sent_at)
		VALUES ($1, NOW() + INTERVAL '48 hours', NULL)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	return err
}

func (b *Bot) reminderWorker() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		b.sendDueReminders()
		<-ticker.C
	}
}

func (b *Bot) sendDueReminders() {
	rows, err := b.db.Query(`
		SELECT user_id
		FROM tg_reminders
		WHERE sent_at IS NULL AND send_at <= NOW()
	`)
	if err != nil {
		log.Printf("[Bot] failed to query due reminders: %v", err)
		return
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			log.Printf("[Bot] failed to scan reminder row: %v", err)
			continue
		}

		if err := b.SendMessage(userID, "Привет, это команда @hsetrackerbot! Просим пройти мини-опрос для улучшения нашего мини-аппа — это займет 2 минуты, но очень поможет сделать бот лучше❤️\n\nСсылка на форму:\nhttps://forms.yandex.ru/cloud/698dae06d046881ae20edfb8/"); err != nil {
			log.Printf("[Bot] failed to send reminder to %d: %v", userID, err)
			continue
		}

		_, err := b.db.Exec(`
			UPDATE tg_reminders
			SET sent_at = NOW()
			WHERE user_id = $1
		`, userID)
		if err != nil {
			log.Printf("[Bot] failed to mark reminder as sent for %d: %v", userID, err)
		}
	}
}

func (b *Bot) SendMessage(userID int64, text string) error {
	if b.api == nil {
		return fmt.Errorf("bot is not initialized")
	}

	recipient := &telebot.Chat{ID: userID}

	log.Printf("[Bot] Sending notification to UserID=%d", userID)
	_, err := b.api.Send(recipient, text, telebot.ModeMarkdown)

	time.Sleep(500 * time.Millisecond)

	return err
}
