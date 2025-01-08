package service

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/initia-labs/weave/common"
	weaveio "github.com/initia-labs/weave/io"
)

type Launchd struct {
	commandName CommandName
}

func NewLaunchd(commandName CommandName) *Launchd {
	return &Launchd{commandName: commandName}
}

func (j *Launchd) GetCommandName() string {
	return string(j.commandName)
}

func (j *Launchd) GetServiceName() (string, error) {
	slug, err := j.commandName.GetServiceSlug()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("com.%s.daemon", slug), nil
}

func (j *Launchd) Create(binaryVersion, appHome string) error {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	weaveDataPath := filepath.Join(userHome, common.WeaveDataDirectory)
	weaveLogPath := filepath.Join(userHome, common.WeaveLogDirectory)
	binaryName, err := j.commandName.GetBinaryName()
	if err != nil {
		return fmt.Errorf("failed to get binary name: %v", err)
	}
	binaryPath := filepath.Join(weaveDataPath, binaryVersion)
	if err = os.Setenv("HOME", userHome); err != nil {
		return fmt.Errorf("failed to set HOME: %v", err)
	}

	serviceName, err := j.GetServiceName()
	if err != nil {
		return fmt.Errorf("failed to get service name: %v", err)
	}
	plistPath := filepath.Join(userHome, fmt.Sprintf("Library/LaunchAgents/%s.plist", serviceName))
	if weaveio.FileOrFolderExists(plistPath) {
		err = weaveio.DeleteFile(plistPath)
		if err != nil {
			return err
		}
	}
	cmd := exec.Command("tee", plistPath)
	template := DarwinTemplateMap[j.commandName]
	cmd.Stdin = strings.NewReader(fmt.Sprintf(string(template), binaryName, binaryPath, appHome, userHome, weaveLogPath, j.GetCommandName()))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}
	return j.reloadService()
}

// func (j *Launchd) unloadService() error {
// 	userHome, err := os.UserHomeDir()
// 	if err != nil {
// 		return fmt.Errorf("failed to get user home directory: %v", err)
// 	}
// 	unloadCmd := exec.Command("launchctl", "unload", filepath.Join(userHome, fmt.Sprintf("Library/LaunchAgents/%s.plist", j.GetServiceName())))
// 	if err = unloadCmd.Run(); err != nil {
// 		return fmt.Errorf("failed to unload service: %v", err)
// 	}
// 	return nil
// }

func (j *Launchd) reloadService() error {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}
	serviceName, err := j.GetServiceName()
	if err != nil {
		return fmt.Errorf("failed to get service name: %v", err)
	}
	unloadCmd := exec.Command("launchctl", "unload", filepath.Join(userHome, fmt.Sprintf("Library/LaunchAgents/%s.plist", serviceName)))
	_ = unloadCmd.Run()
	loadCmd := exec.Command("launchctl", "load", filepath.Join(userHome, fmt.Sprintf("Library/LaunchAgents/%s.plist", serviceName)))
	if err := loadCmd.Run(); err != nil {
		return fmt.Errorf("failed to load service: %v", err)
	}
	return nil
}

func (j *Launchd) Start() error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return fmt.Errorf("failed to get service name: %v", err)
	}
	cmd := exec.Command("launchctl", "start", serviceName)
	return cmd.Run()
}

func (j *Launchd) Stop() error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return fmt.Errorf("failed to get service name: %v", err)
	}
	cmd := exec.Command("launchctl", "stop", serviceName)
	return cmd.Run()
}

func (j *Launchd) Restart() error {
	err := j.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}
	time.Sleep(1 * time.Second)
	err = j.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}
	return nil
}

func (j *Launchd) Log(n int) error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return fmt.Errorf("failed to get service name: %v", err)
	}
	fmt.Printf("Streaming logs from launchd %s\n", serviceName)
	return j.streamLogsFromFiles(n)
}

func (j *Launchd) PruneLogs() error {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	slug, err := j.commandName.GetServiceSlug()
	if err != nil {
		return fmt.Errorf("failed to get service slug: %v", err)
	}
	if err != nil {
		return fmt.Errorf("failed to get service name: %v", err)
	}

	logFilePathOut := filepath.Join(userHome, common.WeaveLogDirectory, fmt.Sprintf("%s.stdout.log", slug))
	logFilePathErr := filepath.Join(userHome, common.WeaveLogDirectory, fmt.Sprintf("%s.stderr.log", slug))

	if err := os.Remove(logFilePathOut); err != nil {
		return fmt.Errorf("failed to remove log file %s: %v", logFilePathOut, err)
	}
	if err := os.Remove(logFilePathErr); err != nil {
		return fmt.Errorf("failed to remove log file %s: %v", logFilePathErr, err)
	}

	return nil
}

// streamLogsFromFiles streams logs from file-based logs
func (j *Launchd) streamLogsFromFiles(n int) error {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	slug, err := j.commandName.GetServiceSlug()
	if err != nil {
		return fmt.Errorf("failed to get service slug: %v", err)
	}
	logFilePathOut := filepath.Join(userHome, common.WeaveLogDirectory, fmt.Sprintf("%s.stdout.log", slug))
	logFilePathErr := filepath.Join(userHome, common.WeaveLogDirectory, fmt.Sprintf("%s.stderr.log", slug))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go j.tailLogFile(logFilePathOut, os.Stdout, n)
	go j.tailLogFile(logFilePathErr, os.Stderr, n)

	<-sigChan

	fmt.Println("Stopping log streaming...")
	return nil
}

func (j *Launchd) tailLogFile(filePath string, output io.Writer, maxLogLines int) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("error opening log file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLogLines {
			lines = lines[1:]
		}
	}

	for _, line := range lines {
		_, _ = fmt.Fprintln(output, line)
	}

	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Printf("error seeking to end of log file %s: %v\n", filePath, err)
		return
	}

	for {
		var line = make([]byte, 4096)
		n, err := file.Read(line)
		if err != nil && err != io.EOF {
			fmt.Printf("error reading log file %s: %v\n", filePath, err)
			return
		}

		if n > 0 {
			_, _ = output.Write(line[:n])
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}
