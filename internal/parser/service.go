package parser

import (
	"fmt"
	"time"

	"github.com/AntiSlang/tracker/internal/storage"
	"github.com/AntiSlang/tracker/internal/tgbot"
)

type Service struct {
	db  *storage.Storage
	bot *tgbot.Bot
}

func New(db *storage.Storage, bot *tgbot.Bot) *Service {
	return &Service{db: db, bot: bot}
}

func (s *Service) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		s.checkAllTables()
	}
}

func (s *Service) checkAllTables() {
	return
}

func (s *Service) fetchAndHash(url string) (string, error) {
	return "", nil
}

func (s *Service) notifySubscribers(table storage.SubjectTable) {
	userIDs, _ := s.db.GetSubscribersForTable(table.ID)
	for _, uid := range userIDs {
		msg := fmt.Sprintf("Обновление в таблице по предмету! \n%s", table.URL)
		err := s.bot.SendMessage(uid, msg)
		if err != nil {
			return
		}
	}
}
