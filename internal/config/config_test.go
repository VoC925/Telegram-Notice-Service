package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	path = "config_test.yml"
)

func TestXxx(t *testing.T) {
	cfg := MustParseConfig(path)
	require.NotNil(t, cfg)
	assert.Equal(t, cfg.Telegram.Token, "token_test")
	assert.Equal(t, cfg.Telegram.ClientID, "client_id_test")
	assert.Equal(t, cfg.Telegram.ClientSecret, "client_secret_test")
	assert.Equal(t, cfg.Telegram.TimePauseRequest, time.Duration(time.Second*40))
	assert.Equal(t, cfg.Telegram.TimeoutUpdate, 59)
	assert.Equal(t, cfg.Telegram.TimeFreshData, time.Duration(time.Second*24))
	assert.Equal(t, cfg.Telegram.Offset, 0)
	assert.Equal(t, cfg.Telegram.IsDebug, true)
	assert.Equal(t, cfg.Api.Timeout, time.Duration(time.Second*20))
	assert.Equal(t, cfg.IsDebug, true)
}
