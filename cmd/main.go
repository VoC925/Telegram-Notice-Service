package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/VoC925/tgBotNotice/internal/api/telegram"
	"github.com/VoC925/tgBotNotice/internal/config"
	"github.com/VoC925/tgBotNotice/pkg/logging"
	"github.com/VoC925/tgBotNotice/pkg/shutdown"
	"github.com/VoC925/tgBotNotice/pkg/utils"
)

const (
	pathCfgFile   = "config.yml" // путь до файла конфигурации
	pathtoLogFile = "app.log"    // путь до файла, в который записываются логи
)

func init() {
	// загрузка конфигурации
	config.MustParseConfig(pathCfgFile)
}

func main() {
	var (
		quitCh = make(chan struct{}) // канал выхода из программы
	)

	// инициализация логера
	if err := initLogger(pathtoLogFile); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	// API телеграм бота
	bot, err := telegram.NewTelegramApi()
	if err != nil {
		slog.With(
			slog.Any("error", err),
		).Error("create bot API")
		os.Exit(1)
	}

	// запуск тг бота
	// горутина для запуска сервиса
	go bot.Start()

	// горутина, слушащая сигнал ОС и завершающая работу сервиса
	go func() {
		if err := shutdown.Shutdown([]os.Signal{os.Interrupt, os.Kill}, bot); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		close(quitCh)
	}()

	<-quitCh
	slog.Debug("bot stopped")
}

// функция для инициализации логера
func initLogger(pathToFile string) error {
	var (
		cfg         *config.Config
		loglevelApp slog.Level
	)

	file, err := utils.OpenFile(pathToFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND)
	if err != nil {
		return fmt.Errorf("%w: %w", fmt.Errorf("open file %s failed", pathToFile), err)
	}

	cfg = config.ConfigInstance
	if cfg.IsDebug {
		loglevelApp = slog.LevelDebug
	} else {
		loglevelApp = slog.LevelInfo
	}
	// кастомный хендлер
	handlerLogger := logging.NewHandlerLogger(
		logging.Production, // стадия разработки
		file,               // вывод логов
		"Bot_ElTechTrade",  // prefix
		&slog.HandlerOptions{
			Level: loglevelApp,
		},
	)
	// сам логгер
	logging.NewSlogLogger(handlerLogger)
	slog.Debug("logger initialized")
	return nil
}
