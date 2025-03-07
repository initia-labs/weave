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
	systemdServiceFilePath     = ".config/systemd/user"
	rootSystemdServiceFilePath = "/etc/systemd/system"
)

type Systemd struct {
	commandName CommandName
	user        *user.User
	userMode    bool
}

func NewSystemd(commandName CommandName) *Systemd {
	currentUser, err := user.Current()
	if err != nil {
		return &Systemd{commandName: commandName}
	}
	return &Systemd{commandName: commandName, user: currentUser, userMode: currentUser.Uid != "0"}
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

// ensureUserServicePrerequisites checks and sets up requirements before any systemd operation
func (j *Systemd) ensureUserServicePrerequisites() error {
	if !j.userMode {
		return nil
	}

	enableCmd := exec.Command("loginctl", "enable-linger", j.user.Username)
	if output, err := enableCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable lingering. Please run 'loginctl enable-linger %s' manually: %v (output: %s)",
			j.user.Username, err, string(output))
	}

	// Check and set XDG_RUNTIME_DIR if not set
	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		uid := j.user.Uid
		runtimeDir := fmt.Sprintf("/run/user/%s", uid)
		if err := os.Setenv("XDG_RUNTIME_DIR", runtimeDir); err != nil {
			return fmt.Errorf("failed to set XDG_RUNTIME_DIR: %v", err)
		}
	}

	// Check and set DBUS_SESSION_BUS_ADDRESS if not set
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		dbusAddr := fmt.Sprintf("unix:path=%s/bus", runtimeDir)
		if err := os.Setenv("DBUS_SESSION_BUS_ADDRESS", dbusAddr); err != nil {
			return fmt.Errorf("failed to set DBUS_SESSION_BUS_ADDRESS: %v", err)
		}
	}

	return nil
}

func (j *Systemd) Create(binaryVersion, appHome string) error {
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

	var userField string
	// root needs to specify the user while --user mode doesn't
	if j.userMode {
		userField = ""
	} else {
		userField = fmt.Sprintf("User=%s\n", j.user.Username)
	}

	if j.userMode {
		// Create user systemd directory if it doesn't exist
		serviceDir := j.getServiceDirPath()
		if err := os.MkdirAll(serviceDir, 0755); err != nil {
			return fmt.Errorf("failed to create systemd user directory: %v", err)
		}
		// Remove sudo and write directly to user's directory
		serviceFile := filepath.Join(serviceDir, serviceName)
		template := LinuxTemplateMap[j.commandName]
		err = os.WriteFile(serviceFile, []byte(fmt.Sprintf(string(template),
			binaryName, binaryPath, string(j.commandName), appHome, userField)), 0644)
		if err != nil {
			return fmt.Errorf("failed to create service file: %v", err)
		}
	} else {
		serviceFile := filepath.Join(j.getServiceDirPath(), serviceName)
		template := LinuxTemplateMap[j.commandName]
		err = os.WriteFile(serviceFile, []byte(fmt.Sprintf(string(template),
			binaryName, binaryPath, string(j.commandName), appHome, userField)), 0644)
		if err != nil {
			return fmt.Errorf("failed to create service file: %v", err)
		}
	}

	if err := j.ensureUserServicePrerequisites(); err != nil {
		return fmt.Errorf("failed to ensure user service prerequisites: %w", err)
	}

	if err = j.daemonReload(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	if err = j.enableService(); err != nil {
		return fmt.Errorf("failed to enable systemd service: %w", err)
	}
	return nil
}

func (j *Systemd) daemonReload() error {
	return j.systemctl("daemon-reload")
}

func (j *Systemd) enableService() error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}

	return j.systemctl("enable", serviceName)
}

func (j *Systemd) systemctl(args ...string) error {
	var cmd *exec.Cmd
	if j.userMode {
		cmd = exec.Command("systemctl", append([]string{"--user"}, args...)...)
	} else {
		cmd = exec.Command("systemctl", args...)
	}

	// Capture both stdout and stderr
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl error: %v, output: %s", err, string(output))
	}
	return nil
}

func (j *Systemd) getServiceDirPath() string {
	if j.userMode {
		userHome, _ := os.UserHomeDir()
		return filepath.Join(userHome, systemdServiceFilePath)
	}
	return rootSystemdServiceFilePath
}

func (j *Systemd) Log(n int) error {
	if err := j.ensureUserServicePrerequisites(); err != nil {
		return err
	}
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	fmt.Printf("Streaming logs from systemd %s\n", serviceName)
	return j.journalctl("-f", "-u", serviceName, "-n", strconv.Itoa(n))
}

func (j *Systemd) journalctl(args ...string) error {
	var cmd *exec.Cmd
	if j.userMode {
		cmd = exec.Command("journalctl", append([]string{"--user"}, args...)...)
	} else {
		cmd = exec.Command("journalctl", args...)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (j *Systemd) PruneLogs() error {
	if err := j.ensureUserServicePrerequisites(); err != nil {
		return err
	}
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	return j.journalctl("--vacuum-time=1s", "--unit", serviceName)
}

func (j *Systemd) Start(optionalArgs ...string) error {
	if err := j.ensureUserServicePrerequisites(); err != nil {
		return err
	}

	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}

	// If we have optional arguments, we need to modify the service file
	if len(optionalArgs) > 0 {
		serviceFile, err := j.GetServiceFile()
		if err != nil {
			return fmt.Errorf("failed to get service file: %v", err)
		}

		// Read the service file
		content, err := os.ReadFile(serviceFile)
		if err != nil {
			return fmt.Errorf("failed to read service file: %w", err)
		}

		// Parse the file line by line to find and modify ExecStart
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "ExecStart=") {
				// Extract existing command and arguments
				parts := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "ExecStart="))

				// Keep binary path and arguments up to --home
				newArgs := make([]string, 0)
				j := 0
				for ; j < len(parts); j++ {
					if strings.HasPrefix(parts[j], "--home") {
						newArgs = append(newArgs, parts[j])
						j++
						break
					}
					newArgs = append(newArgs, parts[j])
				}

				// Add optional arguments
				newArgs = append(newArgs, optionalArgs...)

				// Create new ExecStart line
				lines[i] = "ExecStart=" + strings.Join(newArgs, " ")
				break
			}
		}

		// Write the modified content back to the file
		newContent := strings.Join(lines, "\n")
		if err := os.WriteFile(serviceFile, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write service file: %w", err)
		}

		// Reload daemon to apply changes
		if err := j.daemonReload(); err != nil {
			return fmt.Errorf("failed to reload daemon: %w", err)
		}
	}

	return j.systemctl("start", serviceName)
}

func (j *Systemd) Stop() error {
	if err := j.ensureUserServicePrerequisites(); err != nil {
		return err
	}
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	return j.systemctl("stop", serviceName)
}

func (j *Systemd) Restart() error {
	if err := j.ensureUserServicePrerequisites(); err != nil {
		return err
	}
	serviceName, err := j.GetServiceName()
	if err != nil {
		return err
	}
	return j.systemctl("restart", serviceName)
}

func (j *Systemd) GetServiceFile() (string, error) {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return "", fmt.Errorf("failed to get service name: %v", err)
	}

	return filepath.Join(j.getServiceDirPath(), serviceName), nil
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
	var binary, home string

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
			home = strings.TrimPrefix(line, envPrefix)
			home = strings.Trim(home, `"`)
		}

		if inServiceSection && strings.HasPrefix(line, flagPrefix) {
			parts := strings.Fields(strings.TrimPrefix(line, flagPrefix))
			if len(parts) > 0 {
				binary = parts[0]
			}

			if strings.Contains(line, homeFlag) {
				homeParts := strings.Split(line, homeFlag)
				if len(homeParts) > 1 {
					home = strings.Fields(homeParts[1])[0]
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("failed to scan service file: %v", err)
	}

	if binary == "" {
		return "", "", fmt.Errorf("binary path not found in the service file")
	}

	if j.commandName == Relayer {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("failed to get user home directory: %v", err)
		}

		return binary, filepath.Join(userHome, common.HermesHome), nil
	}

	if home == "" {
		return "", "", fmt.Errorf("home directory not found in the service file")
	}

	return binary, home, nil
}
