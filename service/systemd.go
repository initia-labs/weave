package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/initia-labs/weave/common"
)

const (
	systemdServiceFilePath = "/etc/systemd/system"
)

type Systemd struct {
	commandName CommandName
}

func NewSystemd(commandName CommandName) *Systemd {
	return &Systemd{commandName: commandName}
}

func (j *Systemd) GetCommandName() string {
	return string(j.commandName)
}

func (j *Systemd) GetServiceName() (string, error) {
	slug, err := j.commandName.GetServiceSlug()
	if err != nil {
		return "", fmt.Errorf("failed to get service name: %v", err)
	}
	return slug + ".service", nil
}

func (j *Systemd) Create(binaryVersion, appHome string) error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	binaryName, err := j.commandName.GetBinaryName()
	if err != nil {
		return fmt.Errorf("failed to get current binary name: %v", err)
	}
	var binaryPath string
	switch j.commandName {
	case UpgradableInitia, NonUpgradableInitia:
		binaryPath = filepath.Join(userHome, common.WeaveDataDirectory, binaryVersion)
	case Minitia:
		binaryPath = filepath.Join(userHome, common.WeaveDataDirectory, binaryVersion, strings.ReplaceAll(binaryVersion, "@", "_"))
	default:
		binaryPath = filepath.Join(userHome, common.WeaveDataDirectory)
	}

	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	cmd := exec.Command("sudo", "tee", fmt.Sprintf("%s/%s", systemdServiceFilePath, serviceName))
	template := LinuxTemplateMap[j.commandName]
	cmd.Stdin = strings.NewReader(fmt.Sprintf(string(template), binaryName, currentUser.Username, binaryPath, string(j.commandName), appHome))
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}
	if err = j.daemonReload(); err != nil {
		return err
	}
	return j.enableService()
}

func (j *Systemd) daemonReload() error {
	cmd := exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %v", err)
	}
	return nil
}

func (j *Systemd) enableService() error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	cmd := exec.Command("sudo", "systemctl", "enable", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %v", err)
	}
	return nil
}

func (j *Systemd) Log(n int) error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	fmt.Printf("Streaming logs from systemd %s\n", serviceName)

	cmd := exec.Command("journalctl", "-f", "-u", serviceName, "-n", strconv.Itoa(n))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (j *Systemd) PruneLogs() error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	cmd := exec.Command("journalctl", "--vacuum-time=1s", "--unit", serviceName)
	return cmd.Run()
}

func (j *Systemd) Start() error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	cmd := exec.Command("systemctl", "start", serviceName)
	return cmd.Run()
}

func (j *Systemd) Stop() error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	cmd := exec.Command("systemctl", "stop", serviceName)
	return cmd.Run()
}

func (j *Systemd) Restart() error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	cmd := exec.Command("systemctl", "restart", serviceName)
	return cmd.Run()
}

func (j *Systemd) GetServiceFile() (string, error) {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return "", fmt.Errorf("failed to get service name: %v", err)
	}

	return filepath.Join(systemdServiceFilePath, serviceName), nil
}

func (j *Systemd) GetServiceBinaryAndHome() (string, string, error) {
	serviceFile, err := j.GetServiceFile()
	if err != nil {
		return "", "", fmt.Errorf("failed to get service file: %v", err)
	}

	file, err := os.Open(serviceFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to open service file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inServiceSection := false
	envPrefix := `Environment="DAEMON_HOME=`
	flagPrefix := `ExecStart=`
	homeFlag := "--home "
	var homeValue string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "[Service]" {
			inServiceSection = true
			continue
		}

		if inServiceSection && strings.HasPrefix(line, "[") {
			break
		}

		if inServiceSection && strings.HasPrefix(line, envPrefix) {
			homeValue = strings.TrimPrefix(line, envPrefix)
			homeValue = strings.Trim(homeValue, `"`)
			// TODO: Update this
			return homeValue, "", nil
		}

		if inServiceSection && strings.HasPrefix(line, flagPrefix) {
			if strings.Contains(line, homeFlag) {
				parts := strings.Split(line, homeFlag)
				if len(parts) > 1 {
					homeValue = strings.Fields(parts[1])[0]
					// TODO: Update this
					return homeValue, "", nil
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// TODO: Update this
		return "", "", fmt.Errorf("failed to scan service file: %v", err)
	}

	// TODO: Update this
	return "", "", fmt.Errorf("home directory not found in the service file")
}
