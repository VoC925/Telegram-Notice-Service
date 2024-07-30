package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/VoC925/tgBotNotice/internal/config"
)

// структура access токена
type Token struct {
	Value   string `json:"access_token"` // токен
	Expires int64  `json:"expires_in"`   // время жизни токена в секундах
}

// если время Expires больше, чем текущее время, то токен валиден (true)
func (t Token) IsValid() bool {
	return t.Expires > time.Now().Unix()
}

// структура нового обновления
type UpdateInfo struct {
	Title     string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created"`
}

type UpdateInfoSlice []*UpdateInfo

func (ui UpdateInfoSlice) String() string {
	var str strings.Builder
	for index, elem := range ui {
		str.WriteString(fmt.Sprintf("%d) %s",
			index+1,
			fmt.Sprintf(
				config.UpdateResponseTemplate,
				elem.Title,
				elem.CreatedAt.Format(time.DateTime),
				elem.Path,
			)))
		str.WriteString("\n")
	}
	return str.String()
}

// переопределение метода десереализации
func (ui *UpdateInfoSlice) UnmarshalJSON(data []byte) error {
	var (
		rawData  map[string]*json.RawMessage
		rawItems []*json.RawMessage
		items    []*UpdateInfo
	)
	// первоначально десереализуем все в мапу rawData
	if err := json.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("%w: %w", fmt.Errorf("unmarshal JSON to items struct"), err)
	}

	// десереализуем в слайс "сырых" структур rawItems
	if err := json.Unmarshal(*rawData["items"], &rawItems); err != nil {
		return fmt.Errorf("%w: %w", fmt.Errorf("unmarshal JSON to slice raw"), err)
	}
	// slog.With(slog.Int("len", len(rawItems))).Debug("параметры слайса rawItems")
	// переберем слайс rawItems и десереализуем с слайс структур
	for _, raw := range rawItems {
		updateInfo := &UpdateInfo{}
		if err := json.Unmarshal(*raw, updateInfo); err != nil {
			return fmt.Errorf("%w: %w", fmt.Errorf("unmarshal JSON to UpdateInfo struct"), err)
		}
		items = append(items, updateInfo)
	}
	*ui = items
	return nil
}
