package config

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/VoC925/tgBotNotice/internal/errorApi"
	"github.com/ilyakaznacheev/cleanenv"
)

// структура конфигурации приложения
type Config struct {
	Telegram struct {
		Token            string        `yaml:"token" env-required:"true"`
		ClientID         string        `yaml:"client_id" env-required:"true"`
		ClientSecret     string        `yaml:"client_secret" env-required:"true"`
		TimePauseRequest time.Duration `yaml:"time_pause_request" env-default:"60s"`
		TimeFreshData    time.Duration `yaml:"time_fresh_data" env-default:"60s"`
		TimeoutUpdate    int           `yaml:"timeout_update" env-default:"60s"`
		Offset           int           `yaml:"offset" env-default:"0"`
		IsDebug          bool          `yaml:"is_debug" env-default:"false"`
	} `yaml:"telegram"`
	Server struct {
		Host string `yaml:"host" env-default:"localhost"`
		Port int    `yaml:"port" env-default:"8080"`
	} `yaml:"server"`
	Api struct {
		Timeout time.Duration `yaml:"timeout" env-default:"30s"`
	} `yaml:"api"`
	IsDebug bool `yaml:"is_debug" env-default:"false"`
}

var (
	ConfigInstance *Config
	once           sync.Once
)

// парсинг конфига
func MustParseConfig(pathToCfg string) *Config {
	once.Do(func() {
		ConfigInstance = &Config{}
		err := cleanenv.ReadConfig(pathToCfg, ConfigInstance)
		if err != nil {
			slog.Error(fmt.Errorf("%w: %w", errorApi.ErrParseCfg, err).Error())
			os.Exit(1)
		}
		slog.Debug("Config file read successfully")
	})
	return ConfigInstance
}
