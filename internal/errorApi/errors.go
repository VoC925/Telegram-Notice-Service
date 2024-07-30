package errorApi

import "errors"

var (
	// config
	ErrLoadENV  = errors.New("load .env failed")
	ErrParseCfg = errors.New("parse config failed")
	// bot
	ErrCreateBotApi = errors.New("couldn't create bot api")
	ErrSendMessage  = errors.New("couldn't send message")
	ErrStartAgain   = errors.New("bot started already")
	ErrStop         = errors.New("bot stop failed")
	ErrCtxDeadline  = errors.New("deadline handlind exceeded")
	// command
	ErrStarComand = errors.New("'/start' failed")
	// store
	ErrExpiresToken  = errors.New("token expired")
	ErrTokenNotExist = errors.New("token doesn't exist")
	// авторизация
	ErrDoTokenRequest    = errors.New("token request failed")
	ErrInvalidStatusCode = errors.New("request with status not 200")
	ErrHeaderContentType = errors.New("header Content-Type isn't application/json")
	// запрос к серверу Яндекс
	ErrServiceRequest = errors.New("request service failed")
	ErrUnmarshalJSON  = errors.New("unmarshal JSON")
)
