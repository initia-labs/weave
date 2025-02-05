package service

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Service interface {
	Create(binaryVersion, appHome string) error
	Log(n int) error
	Start() error
	Stop() error
	Restart() error
	PruneLogs() error

	GetServiceFile() (string, error)
	GetServiceBinaryAndHome() (string, string, error)
}

func NewService(commandName CommandName) (Service, error) {
	return NewDocker(commandName), nil
}

func NonDetachStart(s Service) error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalChan)

	go func() {
		err := s.Start()
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
