package config

const (
	// команды
	InfoCmd    = "info"
	AuthCmd    = "auth"     // авторизация
	SendCmd    = "send"     // запуск бота
	StopCmd    = "stop"     // остановка отправки уведомлений бота
	SpecialCmd = "business" // пасхалка-команда
	// состояния авторизации
	NotAuthorized Auth = iota
	Authorized
	// ответы сервера
	RespInfo = `Данный бот выполняет одну функцию - отправка уведомлений.
Уведомления сообщают о добавлении новых файлов на Яндекс Диск.
Например, вы добавили в папку /file на Яндекс Диск файл Info.txt.
В течение нескольких минут вам придет уведомление, сообщающее о появлении нового файла.
Для начала работы необходимо сначала авторизоваться через команду /%s.
Получение уведомлений можно начать командой /%s.
Если вы хотите приостановить получение уведомлений воспользуйтесь командой /%s,
аналогично, при запуске уведомлений - /%s.`
	RespStart             = "Чтение уведомлений успешно запущено"
	RespStop              = "Отправка уведомлений отключена"
	RespStartedAlready    = "Чтение уведомлений уже было запущено"
	RespStopedAlready     = "Чтение уведомлений уже было завершено"
	RespUnknownCmd        = "Данная команда не поддерживается"
	RespOnlyCmd           = "Поддерживаются только команды вида '/(команда)'"
	RespLetsAuth          = "Для начала работы с сервисом необходимо перейти по ссылке ниже"
	RespNeedAuth          = "Для начала работы с сервисом необходимо авторизоваться"
	RespSendCode          = "Введите код, полученный при переходе по ссылке (код действителен 10 минут)"
	RespAuthorizedAlready = "Вы уже были авторизованы ранее"
	RespAuthFail          = "Произошла ошибка авторизации попробуйте снова"
	RespAuthSuccess       = "Авторизация прошла успешно"
	RespTokenFail         = "Возникла внутренняя ошибка. Попробуйте выполнить команду снова"
	// ссылки
	FeatureURL   = `https://www.youtube.com/watch?v=WR9mvNa6FDM#access_token=y0_AgAAAAAIYxaZAAwb5AAAAAEKnHJQAAAasOKqKaZCoLE_95VxCuFIyRKhVQ&token_type=bearer&expires_in=31368557&cid=ahnwb0r94k5uavpykpndj4upc8`
	AuthorizeURL = `https://oauth.yandex.ru/authorize` // url для получение OAuth токена, параметр - значение client_id
	TokenURL     = `https://oauth.yandex.ru/token`
	DiskFilesURL = `https://cloud-api.yandex.net/v1/disk/resources/last-uploaded`
	// ответ пользователю
	UpdateResponseTemplate = `	Название: "%s" 
	Дата добавления: %s
	Путь: "%s"`
)

type Auth int
