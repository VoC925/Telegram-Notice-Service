package yandexdisk

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/VoC925/tgBotNotice/internal/config"
	"github.com/VoC925/tgBotNotice/internal/errorApi"
	"github.com/VoC925/tgBotNotice/internal/models"
)

const (
// таймаут клиента
// timeoutDefault = 5 * time.Second
)

type (
	headers map[string]string // мапа заголовков запроса
	params  map[string]string // мапа параметров запроса
)

type YandexDiskApi interface {
	Update() <-chan *models.UpdateInfoSlice // метод, который отправляет через канал полученные обновления
	Close() error                           // метод для закрытия соедиения с API
	UpdateDiskData(string)                  // метод опрашивающий API
	Stop()                                  // метод останавливающий отправку уведомлений
	// авторизация
	AuthorizeURL() string                            // запросить ссылку для получение кода авторизации
	RequestToken(code string) (*models.Token, error) // получить токен из кода авторизации
}

type yandexDiskAPI struct {
	clientID      string
	clientSecret  string
	client        *http.Client
	pauseRequest  time.Duration // период опроса API
	timeFreshData time.Duration
	updateCh      chan *models.UpdateInfoSlice // канал для отправки обновлений
	stopCh        chan struct{}                // канал для остановки горутины отправки уведомлений
}

// конструктор
func NewYandexDiskAPI() YandexDiskApi {
	cfg := config.ConfigInstance
	return &yandexDiskAPI{
		clientID:     cfg.Telegram.ClientID,
		clientSecret: cfg.Telegram.ClientSecret,
		client: &http.Client{
			Timeout: cfg.Api.Timeout,
		},
		pauseRequest:  cfg.Telegram.TimePauseRequest,
		timeFreshData: cfg.Telegram.TimeFreshData,
		updateCh:      make(chan *models.UpdateInfoSlice),
		stopCh:        make(chan struct{}),
	}
}

// метод возвращает канал для чтения данных
func (api *yandexDiskAPI) Update() <-chan *models.UpdateInfoSlice {
	return api.updateCh
}

func (api *yandexDiskAPI) Close() error {
	close(api.updateCh)
	return nil
}

// метод создает URL ссылку для получения кода авторизации
func (c *yandexDiskAPI) AuthorizeURL() string {
	p := c.createParams(
		params{
			"response_type": "code",
			"client_id":     fmt.Sprint(c.clientID),
		},
	)
	return fmt.Sprintf("%s?%s", config.AuthorizeURL, p)
}

// метод запрашивает токен и добавляет в хранилище
func (c *yandexDiskAPI) RequestToken(code string) (*models.Token, error) {
	return c.requestToken(code)
}

// метод для получения access токена через запрос
func (c *yandexDiskAPI) requestToken(code string) (*models.Token, error) {
	// выполнение запроса
	resp, err := c.doRequest(
		http.MethodPost, // метод запроса
		config.TokenURL, // URL
		strings.NewReader(
			c.createParams(
				params{
					"grant_type":    "authorization_code",
					"code":          code,
					"client_id":     fmt.Sprint(c.clientID),
					"client_secret": c.clientSecret,
				},
			),
		), // тело запроса
		headers{
			"Content-type": "application/x-www-form-urlencoded",
		}, // заголовки
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errorApi.ErrDoTokenRequest, err)
	}
	// валидность ответа
	if err := c.validResponse(resp); err != nil {
		return nil, fmt.Errorf("%w: %w", errorApi.ErrDoTokenRequest, err)
	}
	// получение токена
	return c.parseTokenInfo(resp)
}

// метод, проверяющий валидность ответа
func (c *yandexDiskAPI) validResponse(resp *http.Response) error {
	// статус код ответа не 200
	if resp.StatusCode != http.StatusOK {
		slog.With(slog.Int("code", resp.StatusCode)).Debug("bad status code response")
		return errorApi.ErrInvalidStatusCode
	}
	return nil
}

// метод для парсинга структуры из ответа запроса
func (c *yandexDiskAPI) parseTokenInfo(resp *http.Response) (*models.Token, error) {
	tokenInfo := models.Token{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return &tokenInfo, nil
}

// универсальный метод для выполнения запроса
func (c *yandexDiskAPI) doRequest(method, url string, body io.Reader, head headers) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for key, value := range head {
		req.Header.Add(key, value)
	}
	return c.client.Do(req)
}

// метод, который создает параметры вида "foo=foo&boo=boo"
func (c *yandexDiskAPI) createParams(getParams params) string {
	p := url.Values{}
	for k, v := range getParams {
		p.Set(k, v)
	}
	return p.Encode()
}

func (c *yandexDiskAPI) UpdateDiskData(token string) {
	ticker := time.NewTicker(c.pauseRequest)
	defer ticker.Stop()
	slog.Debug("запущен тикер с отправкой данных в канал")

loop:
	for {
		select {
		case <-ticker.C:
			slog.Debug("сработал тикер")
			// реализация опроса
			// выполнение запроса
			resp, err := c.doRequest(
				http.MethodGet,      // метод запроса
				config.DiskFilesURL, // URL
				nil,
				headers{
					"Authorization": fmt.Sprintf("OAuth %s", token),
				}, // заголовки
			)
			if err != nil {
				slog.With(slog.Any("error", err)).Error("request to service failed")
				continue
			}
			// валидность ответа
			if err := c.validResponse(resp); err != nil {
				slog.With(slog.Any("error", err)).Error("invalid response from service")
				continue
			}
			// парсим ответ в структуру
			updateInfo := &models.UpdateInfoSlice{}
			body := resp.Body
			if err := json.NewDecoder(body).Decode(&updateInfo); err != nil {
				slog.With(slog.Any("error", err)).Error("parse request to service failed")
				body.Close()
				continue
			}
			body.Close()
			// отфильтрованные данные, то есть обновления, которые пришли в течение 1 минуты
			filteredData := c.filter(updateInfo)
			// если нет новых данных, то выходим и ждем нового запроса к сервису
			if len(*filteredData) == 0 {
				slog.Debug("Нет новых данных на Яндекс Диске")
				continue
			}
			// отправляем в канал, если есть что отправлять
			c.updateCh <- filteredData
			slog.Debug("Данные отправлены в канал")
		case <-c.stopCh:
			slog.Debug("Получен сигнал о закрытии канала stopCh")
			break loop
		}
	}
	slog.Debug("UpdateDiskData() закрыта, канал ticker закрыт")
}

func (c *yandexDiskAPI) filter(data *models.UpdateInfoSlice) *models.UpdateInfoSlice {
	var filteredData models.UpdateInfoSlice
	timeNow := time.Now().Add(-1 * c.timeFreshData)
	for _, elem := range *data {
		if timeNow.Before(elem.CreatedAt) {
			filteredData = append(filteredData, elem)
		}
	}
	return &filteredData
}

// метод, который закрывает канал stopCh
func (c *yandexDiskAPI) Stop() {
	close(c.stopCh)
	slog.Debug("канал stopCh закрыт")
	c.stopCh = make(chan struct{})
}
