package service

import (
	"bufio"
	"bytes"
	"encoding/xml"
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

const (
	launchdServiceFilePath = "Library/LaunchAgents"
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
	plistPath := filepath.Join(userHome, launchdServiceFilePath, fmt.Sprintf("%s.plist", serviceName))
	if weaveio.FileOrFolderExists(plistPath) {
		err = weaveio.DeleteFile(plistPath)
		if err != nil {
			return err
		}
	}
	cmd := exec.Command("tee", plistPath)
	template := DarwinTemplateMap[j.commandName]
	cmd.Stdin = strings.NewReader(fmt.Sprintf(string(template), binaryName, binaryPath, appHome, userHome, weaveLogPath, j.GetCommandName()))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create service: %v (output: %s)", err, string(output))
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
	unloadCmd := exec.Command("launchctl", "unload", filepath.Join(userHome, launchdServiceFilePath, fmt.Sprintf("%s.plist", serviceName)))
	_ = unloadCmd.Run()
	loadCmd := exec.Command("launchctl", "load", filepath.Join(userHome, launchdServiceFilePath, fmt.Sprintf("%s.plist", serviceName)))
	if output, err := loadCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load service: %v (output: %s)", err, string(output))
	}
	return nil
}

func (j *Launchd) Start(optionalArgs ...string) error {
	serviceName, err := j.GetServiceName()
	if err != nil {
		return fmt.Errorf("failed to get service name: %v", err)
	}

	plistPath, err := j.GetServiceFile()
	if err != nil {
		return fmt.Errorf("failed to get service file path: %v", err)
	}

	// Read the plist file
	content, err := os.ReadFile(plistPath)
	if err != nil {
		return fmt.Errorf("failed to read plist file: %w", err)
	}

	// Parse the plist XML
	decoder := xml.NewDecoder(bytes.NewReader(content))
	var inProgramArgs bool
	programArgs := make([]string, 0)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to parse plist: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "key" && inProgramArgs {
				inProgramArgs = false
			}
			if t.Name.Local == "array" && inProgramArgs {
				programArgs = []string{}
			}
		case xml.CharData:
			if string(t) == "ProgramArguments" {
				inProgramArgs = true
			}
			if inProgramArgs && len(strings.TrimSpace(string(t))) > 0 {
				programArgs = append(programArgs, strings.TrimSpace(string(t)))
			}
		}
	}

	// Create new arguments list
	newArgs := make([]string, 0)

	fmt.Println("programArgs", programArgs)

	i := 0
	for ; i < len(programArgs); i++ {
		if strings.HasPrefix(programArgs[i], "--home=") {
			newArgs = append(newArgs, programArgs[i])
			i++
			break
		}
		newArgs = append(newArgs, programArgs[i])
	}

	newArgs = append(newArgs, optionalArgs...)
	// Replace program arguments in the original content
	var oldArgsXML, newArgsXML strings.Builder
	for _, arg := range programArgs {
		fmt.Fprintf(&oldArgsXML, "\t\t<string>%s</string>\n", arg)
	}
	for _, arg := range newArgs {
		fmt.Fprintf(&newArgsXML, "\t\t<string>%s</string>\n", arg)
	}

	// Find the ProgramArguments array section and replace its content
	startTag := "<array>"
	endTag := "</array>"
	arrayStart := strings.Index(string(content), startTag)
	arrayEnd := strings.Index(string(content), endTag)
	if arrayStart != -1 && arrayEnd != -1 {
		arrayStart += len(startTag)
		oldContent := string(content[arrayStart:arrayEnd])
		newContent := strings.Replace(string(content), oldContent, "\n"+newArgsXML.String(), 1)

		// Remove debug panic
		// Write back to file
		if err := os.WriteFile(plistPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write plist file: %w", err)
		}
	}

	// Reload and start the service
	if err := j.reloadService(); err != nil {
		return fmt.Errorf("failed to reload service: %w", err)
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

func (j *Launchd) GetServiceFile() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	serviceName, err := j.GetServiceName()
	if err != nil {
		return "", fmt.Errorf("failed to get service name: %v", err)
	}

	return filepath.Join(userHome, launchdServiceFilePath, fmt.Sprintf("%s.plist", serviceName)), nil
}

type Plist struct {
	ProgramArguments           []string `xml:"dict>array>string"`
	EnvironmentVariablesKeys   []string `xml:"dict>dict>key"`
	EnvironmentVariablesValues []string `xml:"dict>dict>string"`
}

func (j *Launchd) GetServiceBinaryAndHome() (string, string, error) {
	serviceFile, err := j.GetServiceFile()
	if err != nil {
		return "", "", fmt.Errorf("failed to get service file: %v", err)
	}

	file, err := os.Open(serviceFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to open plist file: %w", err)
	}
	defer file.Close()

	var plist Plist
	decoder := xml.NewDecoder(file)
	err = decoder.Decode(&plist)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode plist file: %w", err)
	}

	if j.commandName == Relayer {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("failed to get user home directory: %v", err)
		}

		return plist.ProgramArguments[0], filepath.Join(userHome, common.HermesHome), nil
	}

	var home string
	for idx, arg := range plist.ProgramArguments {
		arg = strings.TrimSpace(arg)
		if arg == "--home" {
			home = plist.ProgramArguments[idx+1]
			break
		}
		if strings.HasPrefix(arg, "--home=") {
			home = arg[len("--home="):]
			break
		}
	}

	for idx, key := range plist.EnvironmentVariablesKeys {
		if strings.TrimSpace(key) == "DAEMON_HOME" {
			home = plist.EnvironmentVariablesValues[idx]
			break
		}
	}

	if home == "" {
		return "", "", fmt.Errorf("home directory not found in plist file")
	}

	return plist.ProgramArguments[0], home, nil
}
