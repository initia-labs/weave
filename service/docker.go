package service

import (
	"fmt"
	"os"
	"os/exec"
)

type Docker struct {
	commandName CommandName
}

func NewDocker(commandName CommandName) *Docker {
	return &Docker{
		commandName: commandName,
	}
}

func (d *Docker) Create(version, appHome string) error {
	// Pull the appropriate image based on command type and version
	imageName, err := d.getImageName("main")
	if err != nil {
		return fmt.Errorf("failed to get image name: %v", err)
	}

	pullCmd := exec.Command("docker", "pull", imageName)
	if output, err := pullCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull docker image: %v, output: %s", err, string(output))
	}

	return nil
}

func (d *Docker) Start(optionalArgs ...string) error {
	serviceName, err := d.GetServiceName()
	if err != nil {
		return err
	}

	// // Get binary and home paths
	// _, appHome, err := d.GetServiceBinaryAndHome()
	// if err != nil {
	// 	return err
	// }

	args := []string{
		"run",
		"-d",
		"--name", serviceName,
		"--restart", "unless-stopped",
		"--network", "host",
	}
	args = append(args, optionalArgs...)

	// Add port mappings
	ports, err := d.getPortMappings()
	if err != nil {
		return err
	}
	args = append(args, ports...)

	// Add the image name
	imageName, err := d.getImageName("main")
	if err != nil {
		return err
	}
	args = append(args, imageName)

	// Add command arguments
	cmdArgs, err := d.getCommandArgs()
	if err != nil {
		return err
	}
	args = append(args, cmdArgs...)

	cmd := exec.Command("docker", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start container: %v, output: %s", err, string(output))
	}

	return nil
}

func (d *Docker) Stop() error {
	serviceName, err := d.GetServiceName()
	if err != nil {
		return err
	}

	cmd := exec.Command("docker", "stop", serviceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop container: %v, output: %s", err, string(output))
	}

	// Remove the container after stopping
	rmCmd := exec.Command("docker", "rm", serviceName)
	_ = rmCmd.Run() // Ignore error if container doesn't exist

	return nil
}

func (d *Docker) Log(n int) error {
	serviceName, err := d.GetServiceName()
	if err != nil {
		return err
	}

	cmd := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", n), "-f", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d *Docker) getImageName(version string) (string, error) {
	baseImage := "ghcr.io/initia-labs"

	switch d.commandName {
	case Minitia:
		return fmt.Sprintf("%s/minitiad:%s", baseImage, version), nil
	case UpgradableInitia, NonUpgradableInitia:
		return fmt.Sprintf("%s/initiad:%s", baseImage, version), nil
	case OPinitExecutor:
		return fmt.Sprintf("%s/opinit-bots:%s", baseImage, version), nil
	case OPinitChallenger:
		return fmt.Sprintf("%s/opinit-bots:%s", baseImage, version), nil
	case Relayer:
		return fmt.Sprintf("%s/rapid-relayer:%s", baseImage, version), nil
	default:
		return "", fmt.Errorf("unsupported command: %v", d.commandName)
	}
}

func (d *Docker) getPortMappings() ([]string, error) {
	switch d.commandName {
	case Minitia:
		return []string{
			"-p", "26656:26656", // P2P
			"-p", "26657:26657", // RPC
			"-p", "1317:1317", // REST
			"-p", "8545:8545", // JSON-RPC
			"-p", "9090:9090", // gRPC
		}, nil
	case UpgradableInitia, NonUpgradableInitia:
		return []string{
			"-p", "26656:26656",
			"-p", "26657:26657",
			"-p", "1317:1317",
			"-p", "9090:9090", // gRPC
		}, nil
	case OPinitChallenger:
		return []string{
			"-p", "3000:3000", // REST API
		}, nil
	case OPinitExecutor:
		return []string{
			"-p", "3001:3001", // REST API
		}, nil
	case Relayer:
		return []string{
			"-p", "7010:7010", // REST API
			"-p", "7011:7011", // Telemetry
		}, nil
	default:
		return []string{}, nil
	}
}

func (d *Docker) getVolumeName() (string, error) {
	serviceName, err := d.GetServiceName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-data", serviceName), nil
}

func (d *Docker) getCommandArgs() ([]string, error) {
	switch d.commandName {
	case Minitia:
		return []string{"start", "--home", "/app/data"}, nil
	case UpgradableInitia, NonUpgradableInitia:
		return []string{"start", "--home", "/app/data"}, nil
	case OPinitExecutor:
		return []string{"start", "executor", "--home", "/app/data"}, nil
	case OPinitChallenger:
		return []string{"start", "challenger", "--home", "/app/data"}, nil
	case Relayer:
		return []string{}, nil
	default:
		return nil, fmt.Errorf("unsupported command: %v", d.commandName)
	}
}

func (d *Docker) GetServiceName() (string, error) {
	prettyName, err := d.commandName.GetPrettyName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("weave-%s", prettyName), nil
}

func (d *Docker) GetServiceFile() (string, error) {
	// Not needed for Docker implementation
	return "", nil
}

func (d *Docker) GetServiceBinaryAndHome() (string, string, error) {
	volumeName, err := d.getVolumeName()
	if err != nil {
		return "", "", err
	}
	// Return the volume path as home
	return "", fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName), nil
}

func (d *Docker) PruneLogs() error {
	serviceName, err := d.GetServiceName()
	if err != nil {
		return err
	}

	cmd := exec.Command("docker", "logs", "--truncate", "0", serviceName)
	return cmd.Run()
}

func (d *Docker) Restart() error {
	if err := d.Stop(); err != nil {
		return err
	}
	return d.Start()
}
