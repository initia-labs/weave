package service

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type Service interface {
	Create(binaryVersion, appHome string) error
	Log(n int) error
	Start(optionalArgs ...string) error
	Stop() error
	Restart() error
	PruneLogs() error

	GetServiceFile() (string, error)
	GetServiceBinaryAndHome() (string, string, error)
}

func NewService(commandName CommandName) (Service, error) {
	switch runtime.GOOS {
	case "linux":
		return NewSystemd(commandName), nil
	case "darwin":
		return NewLaunchd(commandName), nil
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func NonDetachStart(s Service) error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalChan)

	go func() {
		err := s.Start("--polling-interval=1000ms")
		if err != nil {
			_ = s.Stop()
			panic(err)
		}
		time.Sleep(1 * time.Second)
		err = s.Log(100)
		if err != nil {
			_ = s.Stop()
			panic(err)
		}
	}()

	<-signalChan
	return s.Stop()
}
