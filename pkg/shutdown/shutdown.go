package shutdown

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

var (
	ErrShutdown = errors.New("ошибка закрытия")
)

// метод, который вызывает метод Close() у интерфейсов при поступлении сигналов
func Shutdown(signals []os.Signal, closeItem ...io.Closer) error {
	signalCh := make(chan os.Signal, 1) // сигнал ОС
	// регистрация сигнала ОС
	signal.Notify(signalCh, signals...)
	// ожидание сигнала
	<-signalCh
	for _, item := range closeItem {
		if err := item.Close(); err != nil {
			return fmt.Errorf("%w: %v", err, item)
		}
	}
	return nil
}
