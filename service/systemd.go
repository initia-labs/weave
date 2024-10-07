package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/initia-labs/weave/utils"
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

func (j *Systemd) GetServiceName() string {
	return string(j.commandName) + ".service"
}

func (j *Systemd) Create(binaryVersion string) error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	weaveDataPath := filepath.Join(userHome, utils.WeaveDataDirectory)
	binaryName := fmt.Sprintf("%sd", j.GetCommandName())
	binaryPath := filepath.Join(weaveDataPath, binaryVersion, strings.ReplaceAll(binaryVersion, "@", "_"))

	cmd := exec.Command("sudo", "tee", fmt.Sprintf("/etc/systemd/system/%s", j.GetServiceName()))
	template := LinuxTemplateMap[j.commandName]
	cmd.Stdin = strings.NewReader(fmt.Sprintf(string(template), binaryName, currentUser.Username, binaryPath))
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
	cmd := exec.Command("sudo", "systemctl", "enable", j.GetServiceName())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %v", err)
	}
	return nil
}

func (j *Systemd) Log(n int) error {
	fmt.Printf("Streaming logs from systemd %s\n", j.GetServiceName())

	cmd := exec.Command("journalctl", "-f", "-u", j.GetServiceName(), "-n", strconv.Itoa(n))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (j *Systemd) Start() error {
	cmd := exec.Command("systemctl", "start", j.GetServiceName())
	return cmd.Run()
}

func (j *Systemd) Stop() error {
	cmd := exec.Command("systemctl", "stop", j.GetServiceName())
	return cmd.Run()
}

func (j *Systemd) Restart() error {
	cmd := exec.Command("systemctl", "restart", j.GetServiceName())
	return cmd.Run()
}
