package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/fatih/color"
)

// дефолтные значения типа stage
const (
	Production stage = iota + 1
	Debugging
)

// тип, описывающий стадию разработки
type stage int

// структура кастомного хендлера
type handlerLogger struct {
	slog.Handler             // хендлер из пакета slog, для подтягивания методов интерфейса
	attr         []slog.Attr // слайс атрибутов, которые появляются при вызове метода With
	output       io.Writer   // интерфейс вывода  логов
	st           stage       // стадия разработки
	prefix       string      // префикс для логов
}

// конструктор логера
func NewHandlerLogger(s stage, output io.Writer, prefix string, opts *slog.HandlerOptions) *handlerLogger {
	var hand slog.Handler
	// выбор формата в зависимости от стадии разработки
	switch s {
	case Production:
		hand = slog.NewTextHandler(output, opts) // вывод логов текстовом виде
	case Debugging:
		hand = slog.NewJSONHandler(output, opts) // вывод логов в формате JSON
	}
	return &handlerLogger{
		Handler: hand,
		output:  output,
		st:      s,
		attr:    nil,
		prefix:  prefix,
	}
}

// Handle() переопределение метода интерфейса slog.Handler
func (a *handlerLogger) Handle(ctx context.Context, r slog.Record) error {
	// если стадия - Production, то вызывается стандартный метод Handle()
	// для хендлера slog.NewJSONHandler
	if a.st == Production {
		return a.Handler.Handle(ctx, r)
	}

	// уровень логирования
	level := r.Level.String()

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	// поля
	var b []byte
	// случай, когда поля заданы через метод With
	if a.attr != nil {
		fields := make(map[string]any, len(a.attr))
		for _, v := range a.attr {
			fields[v.Key] = v.Value.Any()
		}
		bb, err := json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
		b = bb
	} else { // поля заданый через .Info("...", fields)
		fields := make(map[string]any, r.NumAttrs())
		r.Attrs(func(a slog.Attr) bool {
			fields[a.Key] = a.Value.Any()
			return true
		})
		bb, err := json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
		b = bb
	}
	// время + префикс
	timeWithPrefixStr := r.Time.Format(time.DateTime)
	if a.prefix != "" {
		timeWithPrefixStr += fmt.Sprintf(" [%s]", a.prefix)
	}
	// сообщение
	msg := color.CyanString(r.Message)
	// финальный лог
	var res string
	switch {
	case len(b) == 0 || string(b) == "{}":
		res = fmt.Sprintf("%s %s: %s\n", timeWithPrefixStr, level, msg)
	default:
		res = fmt.Sprintf("%s %s: %s\n%s\n", timeWithPrefixStr, level, msg, string(b))
	}
	// обнуляем поле, так как если этого не сделать, то в последующих логах будет вставляться этот атрибут
	a.attr = nil
	// запись в интерфейс
	_, err := a.output.Write([]byte(res))
	return err
}

// WithAttrs() переопределение метода интерфейса slog.Handler для работы с методом With
func (a *handlerLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	hand := a
	hand.attr = attrs
	return hand
}

// конструктор логгера
func NewSlogLogger(hand *handlerLogger) {
	// логер с кастомным логером
	l := slog.New(hand)
	// установка кастомного логера в качестве дефолтного для пакета slog
	slog.SetDefault(l)
}
