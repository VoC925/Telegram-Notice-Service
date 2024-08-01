package store

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/VoC925/tgBotNotice/internal/errorApi"
	"github.com/VoC925/tgBotNotice/internal/models"
)

var _ Storer = new(storeToken)

// интерфейс хранилища токена, состоит из мапы, которая хранит access токены сессии
type Storer interface {
	AddAccessToken(int64, *models.Token) // добавление в хранилище нового токена
	ReturnAccessToken(int64) (*models.Token, error)
}

// структура
type storeToken struct {
	mu   *sync.RWMutex           // мьютекс для безопасной, конкуретной работы с мапой cash
	cash map[int64]*models.Token // хранилище токеной, где ключ - chatID
}

// конструктор
func NewStoreToken() Storer {
	slog.Info("создано хранилище токенов")
	return &storeToken{
		mu:   &sync.RWMutex{},
		cash: make(map[int64]*models.Token),
	}
}

// метод добавления нового токена
func (s *storeToken) AddAccessToken(chatID int64, token *models.Token) {
	s.mu.Lock()
	s.cash[chatID] = token
	s.mu.Unlock()
	slog.Info(fmt.Sprintf("chat_id:%v; added new token", chatID))
}

// метод возвращает токен из хранилища
func (s *storeToken) ReturnAccessToken(chatID int64) (*models.Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.cash[chatID]
	// токена нет
	if !ok {
		return nil, errorApi.ErrTokenNotExist
	}
	return token, nil
}
