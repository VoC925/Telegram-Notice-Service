package telegram

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/VoC925/tgBotNotice/internal/api/store"
	"github.com/VoC925/tgBotNotice/internal/api/yandexdisk"
	"github.com/VoC925/tgBotNotice/internal/config"
	"github.com/VoC925/tgBotNotice/internal/errorApi"
	"github.com/VoC925/tgBotNotice/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// структура API telegram бота
type TelegramApi struct {
	botApi   *tgbotapi.BotAPI
	endpoint string

	yandexApi yandexdisk.YandexDiskApi // интерфейс API Яндекс Диска

	stopNoticeCh chan struct{}           // канал для остановки чтения уведомлений
	updateCh     tgbotapi.UpdatesChannel // канал чтения сообщений от пользователя самого бота
	authCodeCh   chan string             // канал для передачи кода автторизации

	isSending    bool         // состояние чтения уведомлений: true - читает, false - не читает
	isAuthState  bool         // состояние авторизации: true - пользователю отправлена ссылка авторизации
	tokenStorage store.Storer // мапа хранилище токенов
}

// конструктор структуры TelegramApi
func NewTelegramApi() (*TelegramApi, error) {
	// берем конфиг
	cfg := config.ConfigInstance

	tgApi := &TelegramApi{
		endpoint:     tgbotapi.APIEndpoint,
		isSending:    false,
		isAuthState:  false,
		stopNoticeCh: make(chan struct{}),
		authCodeCh:   make(chan string),
	}

	bot, err := tgbotapi.NewBotAPIWithClient(
		cfg.Telegram.Token,
		tgApi.endpoint,
		// клиент для telegram API
		&http.Client{
			Timeout: cfg.Api.Timeout,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errorApi.ErrCreateBotApi, err)
	}
	tgApi.botApi = bot

	// API для яндекс диска
	tgApi.setYandexApi(yandexdisk.NewYandexDiskAPI())

	// хранилище токенов
	tgApi.tokenStorage = store.NewStoreToken()

	// уровень debug
	tgApi.botApi.Debug = cfg.Telegram.IsDebug

	// настройка updates
	tgApi.setUpdates()

	slog.With(
		slog.String("bot_account", tgApi.botApi.Self.UserName),
		slog.String("timeout_client", cfg.Api.Timeout.String()),
		slog.Int("timeout_update", cfg.Telegram.TimeoutUpdate),
	).Info("bot created")
	return tgApi, nil
}

// метод устанавливает значение интерфейса API Яндекс Диска
func (tg *TelegramApi) setYandexApi(api yandexdisk.YandexDiskApi) *TelegramApi {
	tg.yandexApi = api
	return tg
}

// установка updates
func (tg *TelegramApi) setUpdates() *TelegramApi {
	cfg := config.ConfigInstance
	u := tgbotapi.NewUpdate(cfg.Telegram.Offset)
	u.Timeout = cfg.Telegram.TimeoutUpdate
	tg.updateCh = tg.botApi.GetUpdatesChan(u)
	return tg
}

// метод возвращет канал, который слушает апдейты
func (tg *TelegramApi) updateEventCh() tgbotapi.UpdatesChannel {
	// tgbotapi.
	return tg.updateCh
}

// метод запуска телеграм бота
func (tg *TelegramApi) Start() {
	slog.Info("bot working started succesfully")
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

// метод для обработки сообщений
func (tg *TelegramApi) handleMsg(msg *tgbotapi.Message) error {
	// контекст для запроса к API
	chatID := msg.Chat.ID
	// slog.With(slog.Any("value", chatID)).Debug("chatID")
	// пришла команда
	if msg.IsCommand() {
		// обнуление состояния авторизации, если ранее пользователь запустил процесс авторизации
		// и вместо того, чтобы ввести код авторизации ввел новую команду
		if tg.isAuthState {
			tg.isAuthState = false
		}
		// команды не требующие авторизации
		switch msg.Command() {
		case config.AuthCmd:
			tg.authorize(chatID)
			return nil
		case config.InfoCmd:
			tg.info(chatID)
			return nil
		default:
		}
		// команды требующие авторизации пользователя
		if tg.isAuthorized(chatID) {
			switch msg.Command() {
			case config.SpecialCmd:
				tg.specialFeature(chatID)
				return nil
			// старт чтения уведомлений
			case config.SendCmd:
				// запускается в горутине, так как метод блокирующий
				tg.startSendNotice(chatID)
				return nil
				// остановка чтения уведомления
			case config.StopCmd:
				if !tg.stopSendNotice() {
					tg.sendMsg(chatID, config.RespStopedAlready)
				}
				tg.sendMsg(chatID, config.RespStop)
				return nil
				// пасхалка
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
	if tg.isAuthState { // если ожидается код авторизации
		// отправляем код в канал
		tg.authCodeCh <- msg.Text
		tg.isAuthState = false
		return nil
	}
	tg.sendMsg(msg.Chat.ID, config.RespOnlyCmd)
	return nil
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

// метод закрывает канал и останаваливает получение новых уведомлений
func (tg *TelegramApi) stopSendNotice() bool {
	// если не было запущено чтение уведомлений, то выходим
	if !tg.isSending {
		return false
	}
	close(tg.stopNoticeCh)
	slog.Info("канал stopNoticeCh для чтения закрыт")
	// так как канал закрыт, то он кидает бесконечно новые структуры
	// необходимо перезаписать значение канала для того, чтобы была
	// возможность заново запускать отправку новых уведомлений
	tg.stopNoticeCh = make(chan struct{})
	tg.isSending = false
	return true
}

// метод для отправки сообщений об ошибке
// chatID - ID чата, куда отправить
// msg - сообщение
func (tg *TelegramApi) sendMsg(chatID int64, msg string) {
	response := tgbotapi.NewMessage(chatID, msg)
	tg.botApi.Send(response)
}

// метод запускает отправку новых уведомлений
func (tg *TelegramApi) startSendNotice(chatID int64) {
	// проверка на уже запущенное чтение
	if tg.isSending {
		slog.Info(fmt.Sprintf("chat_id: %v; получение уведомлений уже было запущено", chatID))
		tg.sendMsg(chatID, config.RespStartedAlready)
		return
	}
	tg.isSending = true
	// метод отправляет уведомления в канал интерфейса в горутине
	// берем токен из хранилища
	t, err := tg.tokenStorage.ReturnAccessToken(chatID)
	if err != nil {
		slog.Info(fmt.Sprintf("chat_id: %v; ошибка получения токена метод startSendNotice()", chatID))
		tg.sendMsg(chatID, config.RespTokenFail)
		return
	}
	go tg.yandexApi.UpdateDiskData(t.Value)
	tg.sendMsg(chatID, config.RespStart)

loop:
	for {
		select {
		// чтение из канала уведомлений
		case data := <-tg.yandexApi.Update():
			slog.Info(fmt.Sprintf("chat_id: %v; данные получены из канала", chatID))
			tg.sendMsg(chatID, models.UpdateInfoSlice(data).String())
			// чтение из канала остановки отправки уведомлений
		case <-tg.stopNoticeCh:
			slog.Info(fmt.Sprintf("chat_id: %v; получен сигнал о закрытии канала stopNoticeCh", chatID))
			// надо закрыть канал и выйти из горутины UpdateDiskData()
			tg.yandexApi.Stop()
			break loop
		}
	}
	slog.Debug("выход из метода startSendNotice()")
}

// авторизация
func (tg *TelegramApi) authorize(chatID int64) {
	// если пользователь авторизован, то есть токен в хранилище
	t, err := tg.tokenStorage.ReturnAccessToken(chatID)
	// если токена нет
	if errors.Is(err, errorApi.ErrTokenNotExist) {
		slog.Info(fmt.Sprintf("chat_id: %v; токена нет в хранилище", chatID))
		tg.handleAuth(chatID)
		return
	}
	// если токен  протух
	if t.IsValid() {
		slog.Debug("токен протух")
		tg.handleAuth(chatID)
		return
	}
	slog.Info(fmt.Sprintf("chat_id: %v; токен есть в хранилище и он валиден", chatID))
	// если токен есть в хранилище и он валиден
	tg.sendMsg(chatID, config.RespAuthorizedAlready)
}

func (tg *TelegramApi) handleAuth(chatID int64) {
	// состояние авторизации
	tg.isAuthState = true
	// если токена нет, значит отправляем пользователю ссылку авторизации
	tg.sendMsg(chatID, fmt.Sprintf("%s:\n%s", config.RespLetsAuth, tg.yandexApi.AuthorizeURL()))
	tg.sendMsg(chatID, config.RespSendCode)
	// слушаем канал и ожидаем код подтверждения
	for code := range tg.authCodeCh {
		token, err := tg.yandexApi.RequestToken(code)
		if err != nil {
			slog.Error(err.Error())
			tg.sendMsg(chatID, config.RespAuthFail)
			return
		}
		// сохраняем новый токен в хранилище
		tg.tokenStorage.AddAccessToken(chatID, token)
		tg.sendMsg(chatID, config.RespAuthSuccess)
	}
}

func (tg *TelegramApi) isAuthorized(chatID int64) bool {
	t, err := tg.tokenStorage.ReturnAccessToken(chatID)
	// если токена нет
	if errors.Is(err, errorApi.ErrTokenNotExist) {
		return false
	}
	// если токен протух
	if t.IsValid() {
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
	tg.botApi.StopReceivingUpdates()
	return nil
}
