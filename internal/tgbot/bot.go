package tgbot

import (
	"fmt"
	"log"
	"time"

	"github.com/AntiSlang/tracker/internal/storage"
	"gopkg.in/telebot.v3"
)

type Bot struct {
	token string
	db    *storage.Storage
	api   *telebot.Bot
}

func New(token string, db *storage.Storage) *Bot {
	return &Bot{
		token: token,
		db:    db,
	}
}

func (b *Bot) Start() error {
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

		newUser := storage.User{
			ID:                   user.ID,
			FirstName:            user.FirstName,
			LastName:             user.LastName,
			MiddleName:           user.Username,
			ProgramID:            nil,
			CourseNumber:         1,
			Language:             user.LanguageCode,
			NotificationsEnabled: true,
		}

		err := b.db.UpsertUser(newUser)
		if err != nil {
			log.Printf("Failed to register user %d: %v", user.ID, err)
			return c.Send("Произошла ошибка при регистрации. Попробуйте позже.")
		}

		log.Printf("User registered: %d (%s)", user.ID, fullName)
		return c.Send(fmt.Sprintf("Привет, %s! Я бот для отслеживания оценок в Вышке. Нажми кнопку, чтобы перейти в мини-приложение", fullName))
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
