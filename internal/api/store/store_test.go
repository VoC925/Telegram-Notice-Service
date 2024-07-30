package store

import (
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/VoC925/tgBotNotice/internal/errorApi"
	"github.com/VoC925/tgBotNotice/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// возвращает токен валидный, либо невалидный
func getToken(isValid bool) *models.Token {
	var currentYear int // текущий год
	if !isValid {
		currentYear = 2022
	} else {
		currentYear = time.Now().Year() + 1
	}
	return &models.Token{
		Value:   "test_string_token",
		Expires: time.Date(currentYear, 1, 1, 0, 0, 0, 0, time.Local).Unix(),
	}
}

func TestAddAccessTokenOk(t *testing.T) {
	var (
		chatID     = int64(rand.Intn(1_000_000))
		validToken = getToken(true)
	)
	store := NewStoreToken()
	store.AddAccessToken(chatID, validToken)
	gotToken, err := store.ReturnAccessToken(chatID)
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(validToken, gotToken))
}

func TestReturnTokenFail(t *testing.T) {
	var (
		chatID = int64(rand.Intn(1_000_000))
	)
	store := NewStoreToken()
	gotToken, err := store.ReturnAccessToken(chatID)
	require.ErrorIs(t, err, errorApi.ErrTokenNotExist)
	assert.Nil(t, gotToken)
}

func TestAddAccessTokenParallel(t *testing.T) {
	var (
		numWorkers = 20
		numIDs     = 10
		ch         = make(chan struct {
			token  *models.Token
			chatID int64
		}, 10)
	)
	// доступ именно к структуре хранилища без интерфейса
	store := &storeToken{
		mu:   &sync.RWMutex{},
		cash: make(map[int64]*models.Token),
	}
	// запускаем воркеров
	for i := 0; i < numWorkers; i++ {
		go func(num int) {
			for j := 0; j < numIDs; j++ {
				ch <- struct {
					token  *models.Token
					chatID int64
				}{
					token:  getToken(true),
					chatID: int64(rand.Intn(1_000_000)),
				}
			}
		}(numIDs)
	}

	go func() {
		for data := range ch {
			store.AddAccessToken(data.chatID, data.token)
		}
	}()
	time.Sleep(1 * time.Second)
	assert.Equal(t, len(store.cash), numWorkers*numIDs)
}

func TestAddAccessTokenRewrite(t *testing.T) {
	var (
		chatID   = int64(rand.Intn(1_000_000))
		tokenOld = getToken(false)
		tokenNew = getToken(true)
	)
	store := NewStoreToken()
	store.AddAccessToken(chatID, tokenOld)
	store.AddAccessToken(chatID, tokenNew)
	tokenGot, err := store.ReturnAccessToken(chatID)
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(tokenGot, tokenNew))
}
