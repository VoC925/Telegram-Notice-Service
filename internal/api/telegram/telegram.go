package telegram

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/VoC925/tgBotNotice/internal/api/yandexdisk"
	"github.com/VoC925/tgBotNotice/internal/config"
	"github.com/VoC925/tgBotNotice/internal/errorApi"
	"github.com/VoC925/tgBotNotice/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// структура API telegram бота
type TelegramApi struct {
	bot *tgbotapi.BotAPI // структура телеграмм бота

	yandexApi yandexdisk.YandexDiskApi // интерфейс API Яндекс Диска

	updateCh   tgbotapi.UpdatesChannel // канал чтения сообщений от пользователя самого бота
	authCodeCh chan string             // канал для передачи кода авторизации

	admin       string        // никнейм админа
	isAuthState bool          // состояние авторизации одно (только админ): true - пользователю отправлена ссылка авторизации
	token       *models.Token // access токен

	mu        sync.RWMutex   // мьютекс для мапы isSending
	listeners map[int64]bool // состояние чтения уведомлений для каждого отдельного чата: true - читает, false - не читает
}

// конструктор структуры TelegramApi
func NewTelegramApi() (*TelegramApi, error) {
	// берем конфиг
	cfg := config.ConfigInstance

	tgApi := &TelegramApi{
		listeners:   make(map[int64]bool),
		isAuthState: false,
		authCodeCh:  make(chan string),
		token:       nil,
		mu:          sync.RWMutex{},
	}

	bot, err := tgbotapi.NewBotAPIWithClient(
		cfg.Telegram.Token,
		tgbotapi.APIEndpoint,
		// клиент для telegram API
		&http.Client{
			Timeout: cfg.Api.Timeout,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errorApi.ErrCreateBotApi, err)
	}
	tgApi.bot = bot

	// API для яндекс диска
	tgApi.yandexApi = yandexdisk.NewYandexDiskAPI()

	// уровень debug
	tgApi.bot.Debug = cfg.Telegram.IsDebug

	// настройка updates
	u := tgbotapi.NewUpdate(cfg.Telegram.Offset)
	u.Timeout = cfg.Telegram.TimeoutUpdate
	tgApi.updateCh = tgApi.bot.GetUpdatesChan(u)

	// установка админа
	tgApi.admin = cfg.Telegram.Admin

	slog.With(
		slog.String("bot_account", tgApi.bot.Self.UserName),
		slog.String("timeout_client", cfg.Api.Timeout.String()),
		slog.Int("timeout_update", cfg.Telegram.TimeoutUpdate),
	).Info("bot created")
	return tgApi, nil
}

// метод запуска телеграм бота
func (tg *TelegramApi) Start() {
	slog.Info("bot working started succesfully")
	// метод отправляющий
	tg.listenUpdates()
}

// метод для обработки приходящих апдейтов
func (tg *TelegramApi) listenUpdates() {
	// чтение из канала updates
	for update := range tg.updateEventCh() {
		if update.Message != nil { // If we got a message
			slog.Info(fmt.Sprintf("chat_id: %v; user: %s; msg receieved: %s",
				update.Message.Chat.ID,
				update.Message.From.UserName,
				update.Message.Text,
			))
			// логика обработки сообщения в горутине, метод обрабатывается в горутине
			// так как чтение из канала уведомлений является блокирующей операцией, то
			// необходимо продолжать читать сообщения полльзователя
			go func() {
				if err := tg.handleMsg(update.Message); err != nil {
					slog.With(
						slog.Any("chatID", update.Message.Chat.ID),
					).Error(err.Error())
				}
			}()
		}
	}
}

// метод возвращет канал, который слушает апдейты
func (tg *TelegramApi) updateEventCh() tgbotapi.UpdatesChannel {
	// tgbotapi.
	return tg.updateCh
}

// метод для обработки сообщений
func (tg *TelegramApi) handleMsg(msg *tgbotapi.Message) error {
	// контекст для запроса к API
	chatID := msg.Chat.ID
	from := msg.From.UserName
	// пришла команда
	if msg.IsCommand() {
		// обнуление состояния авторизации, если ранее админ запустил процесс авторизации
		// и вместо того, чтобы ввести код авторизации ввел новую команду
		// команда только для админа
		if tg.isAuthState && tg.isAdmin(from) {
			tg.isAuthState = false
		}
		// команды не требующие авторизации
		switch msg.Command() {
		case config.AuthCmd:
			// команда только для админа
			tg.auth(chatID, from)
			return nil
		case config.InfoCmd:
			tg.info(chatID)
			return nil
		case config.SpecialCmd:
			tg.specialFeature(chatID)
			return nil
		case config.DeleteListeners:
			tg.deleteAllListeners(chatID, from)
			return nil
		default:
			// tg.sendMsg(chatID, config.RespUnknownCmd)
			// return nil
		}
		// команды требующие авторизации пользователя
		if tg.isAuthorized() {
			switch msg.Command() {
			case config.SendCmd:
				// старт чтения уведомлений
				tg.startSendNotice(chatID)
				return nil
			case config.StopCmd:
				// остановка чтения уведомления
				tg.stopSendNotice(chatID)
				return nil
			default:
				// случай, если пользователь отправил не известную команду
				tg.sendMsg(chatID, config.RespUnknownCmd)
				return nil
			}
		} else {
			tg.sendMsg(chatID, config.RespNeedAuth)
			return nil
		}
	}
	// ответ, если пришла не команда, а просто сообщение
	if tg.isAuthState && tg.isAdmin(from) { // если ожидается код авторизации от админа
		// отправляем код в канал
		tg.authCodeCh <- msg.Text
		tg.isAuthState = false
		return nil
	}
	tg.sendMsg(msg.Chat.ID, config.RespOnlyCmd)
	return nil
}

func (tg *TelegramApi) auth(chatID int64, from string) {
	if tg.token != nil {
		tg.sendMsg(chatID, config.RespAuthorizedAlready)
		return
	}
	if tg.isAdmin(from) {
		tg.authorize(chatID)
		return
	}
	tg.sendMsg(chatID, config.RespOnlyAdmin)
}

// метод отправляет всем слушателям из мапы listener данные
func (tg *TelegramApi) sendToListeners(data *models.UpdateInfoSlice) {
	tg.mu.RLock()
	if len(tg.listeners) == 0 {
		// если пока нет слушателей, то выходим
		tg.mu.RUnlock()
		return
	}
	for chatID, value := range tg.listeners {
		if value {
			// если chat_id имеет состояние true на чтении
			tg.sendMsg(chatID, data.String())
		}
	}
	tg.mu.RUnlock()
}

// метод удаляющий всех слушателей, кроме самого админа
// команда только для админа
func (tg *TelegramApi) deleteAllListeners(chatID int64, from string) {
	// проверка, админ ли отправил команду
	if tg.isAdmin(from) {
		tg.mu.Lock()
		for key := range tg.listeners {
			if key == chatID {
				// chat_id принадлежит админу
				continue
			}
			// удаляем пару ключ-значение
			delete(tg.listeners, key)
		}
		tg.mu.Unlock()
		slog.Info("Все слушатели удалены из мапы listeners")
		return
	}
	tg.sendMsg(chatID, config.RespOnlyAdmin)
}

// метод возвращает какое состояние чтения у заданного chatID
// если chatID нет в мапе, то возвращает ошибка
func (tg *TelegramApi) listenerState(chatID int64) (bool, error) {
	tg.mu.RLock()
	state, ok := tg.listeners[chatID]
	if !ok {
		tg.mu.RUnlock()
		slog.Info(fmt.Sprintf("chat_id: %v; нет в мапе listener", chatID))
		return false, errorApi.ErrNoListener
	}
	tg.mu.RUnlock()
	return state, nil
}

// метод добавляющий новый chat_id в качестве нового listener
func (tg *TelegramApi) addListener(chatID int64) {
	tg.mu.Lock()
	tg.listeners[chatID] = false
	tg.mu.Unlock()
	slog.Info(fmt.Sprintf("chat_id: %v; добавлен в мапу listeners", chatID))
}

// метод изменяет состояние чтения
func (tg *TelegramApi) changeStateListener(chatID int64, state bool) {
	tg.mu.Lock()
	tg.listeners[chatID] = state
	tg.mu.Unlock()
	slog.Info(fmt.Sprintf("chat_id: %v; изменено состояние на %t", chatID, state))
}

// метод проверяет является ли пользователь админом
func (tg *TelegramApi) isAdmin(from string) bool {
	return from == tg.admin
}

func (tg *TelegramApi) info(chatID int64) {
	tg.sendMsg(chatID,
		fmt.Sprintf(config.RespInfo,
			config.AuthCmd,
			config.SendCmd,
			config.StopCmd,
			config.SendCmd,
		),
	)
}

// метод добавляет новый client_id в мапу listener и меняет его состояние на true
// true - готов слушать
func (tg *TelegramApi) startSendNotice(chatID int64) {
	state, err := tg.listenerState(chatID)
	if state && err == nil {
		// проверка на уже запущенное чтение
		// если state = true, то есть уже запущено чтение
		tg.sendMsg(chatID, config.RespStartedAlready)
		return
	}
	// добавляем новый client_id в мапу listener
	tg.addListener(chatID)
	// изменяем состояние чтения на true
	tg.changeStateListener(chatID, true)
	tg.sendMsg(chatID, config.RespStart)
}

// метод меняет состояние client_id в мапе listeners на false
func (tg *TelegramApi) stopSendNotice(chatID int64) {
	state, err := tg.listenerState(chatID)
	if errors.Is(err, errorApi.ErrNoListener) {
		// если client_id нет в мапе
		tg.sendMsg(chatID, config.RespStopedFirstly)
		return
	}
	if !state && err != nil {
		// проверка на уже остановленное чтение
		// если state = false, то есть уже остановлено чтение
		tg.sendMsg(chatID, config.RespStopedAlready)
		return
	}
	// изменяем состояние чтения на false
	tg.changeStateListener(chatID, false)
	slog.Debug(fmt.Sprintf("client_id: %v; остановлено чтение", chatID))
	tg.sendMsg(chatID, config.RespStop)
}

// метод для отправки уведомлений всем слушателям из мапы listeners
func (tg *TelegramApi) sendingLoop() {
	// метод отправляющий уведомления в канал путем зпросов к API
	go tg.yandexApi.UpdateDiskData(tg.token.Value)
	// слушаем канал уведомлений
	for data := range tg.yandexApi.Update() {
		tg.sendToListeners(data)
	}
	slog.Info("выход из метода startSendNotice()")
}

// метод для отправки сообщений об ошибке
// chatID - ID чата, куда отправить
// msg - сообщение
func (tg *TelegramApi) sendMsg(chatID int64, msg string) {
	response := tgbotapi.NewMessage(chatID, msg)
	tg.bot.Send(response)
}

// авторизация доступна только один раз
func (tg *TelegramApi) authorize(chatID int64) {
	// если токена нет значит админ не авторизовался
	if tg.token == nil {
		tg.handleAuth(chatID)
		return
	}
	// если токен протух
	if !tg.token.IsValid() {
		slog.Info("токен протух")
		tg.handleAuth(chatID)
		return
	}
	slog.Info(fmt.Sprintf("chat_id: %v; токен есть и он валиден", chatID))
	// если токен есть и он валиден
	tg.sendMsg(chatID, config.RespAuthorizedAlready)
}

func (tg *TelegramApi) handleAuth(chatID int64) {
	// состояние авторизации у админа
	tg.isAuthState = true
	// если токена нет, значит отправляем пользователю ссылку авторизации
	tg.sendMsg(chatID, fmt.Sprintf("%s:\n%s", config.RespLetsAuth, tg.yandexApi.AuthorizeURL()))
	tg.sendMsg(chatID, config.RespSendCode)
	// слушаем канал и ожидаем код подтверждения
	for code := range tg.authCodeCh {
		t, err := tg.yandexApi.RequestToken(code)
		if err != nil {
			slog.Error(err.Error())
			tg.sendMsg(chatID, config.RespAuthFail)
			return
		}
		// сохраняем токен
		tg.token = t
		slog.Info("Добавлен новый access токен")
		tg.sendMsg(chatID, config.RespAuthSuccess)

		// запуск чтения из Api
		tg.sendingLoop()
	}
}

func (tg *TelegramApi) isAuthorized() bool {
	// если токена нет
	if tg.token == nil {
		return false
	}
	// если токен протух
	if tg.token.IsValid() {
		return false
	}
	return true
}

func (tg *TelegramApi) specialFeature(chatID int64) {
	msg := fmt.Sprintf("Для получения подробной информации перейдите по ссылке ниже:\n%s", config.FeatureURL)
	tg.sendMsg(chatID, msg)
}

// метод закрывающий канал update
func (tg *TelegramApi) Close() error {
	slog.Info("stop listening update chanel")
	tg.bot.StopReceivingUpdates()
	return nil
}
