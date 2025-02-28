package service

import "os/exec"

type Service interface {
	Create(appHome, customDockerImage string) error
	Log(n int) error
	Start(detach bool) error
	Stop() error
	Restart() error
	PruneLogs() error
	RunCmd(options []string, args ...string) *exec.Cmd
}

func NewService(cmd Command) (Service, error) {
	return NewDocker(cmd), nil
}

// func NonDetachStart(s Service) error {
// 	signalChan := make(chan os.Signal, 1)
// 	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
// 	defer signal.Stop(signalChan)

// 	go func() {
// 		err := s.Start()
// 		if err != nil {
// 			_ = s.Stop()
// 			panic(err)
// 		}
// 		time.Sleep(1 * time.Second)
// 		err = s.Log(100)
// 		if err != nil {
// 			_ = s.Stop()
// 			panic(err)
// 		}
// 	}()

// 	<-signalChan
// 	return s.Stop()
// }
