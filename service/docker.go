package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"

	"github.com/initia-labs/weave/common"
)

type Docker struct {
	commandName    CommandName
	vmType         string
	relayerVersion string // Cached version for relayer to ensure Create() and Start() use the same version
}

func NewDocker(commandName CommandName, vmType string) *Docker {
	return &Docker{
		commandName: commandName,
		vmType:      vmType,
	}
}

func (d *Docker) Create(version, appHome string) error {
	// For Rollytics, use docker compose
	if d.commandName == Rollytics {
		return d.createDockerCompose(version)
	}

	// For relayer, fetch and cache the version to ensure Create() and Start() use the same version
	if d.commandName == Relayer {
		d.relayerVersion = GetRapidRelayerVersion()
		version = d.relayerVersion
	}

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	// Pull the appropriate image based on command type and version
	imageName, err := d.getImageName(version)
	if err != nil {
		return fmt.Errorf("failed to get image name: %v", err)
	}

	out, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull docker image: %v", err)
	}
	defer out.Close()

	// Read and discard pull output (or could display progress)
	if _, err := io.Copy(io.Discard, out); err != nil {
		return fmt.Errorf("failed to read pull output: %v", err)
	}

	// Create named volume for data persistence
	volumeName, err := d.getVolumeName()
	if err != nil {
		return fmt.Errorf("failed to get volume name: %v", err)
	}

	_, err = cli.VolumeCreate(ctx, volume.CreateOptions{
		Name: volumeName,
	})
	if err != nil {
		return fmt.Errorf("failed to create docker volume: %v", err)
	}

	return nil
}

func (d *Docker) Start(optionalArgs ...string) error {
	// For Rollytics, use docker compose
	if d.commandName == Rollytics {
		return d.startDockerCompose()
	}

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	serviceName, err := d.GetServiceName()
	if err != nil {
		return err
	}

	// Get volume name for data persistence
	volumeName, err := d.getVolumeName()
	if err != nil {
		return err
	}

	// Get image name
	version := "main"
	if d.commandName == Relayer {
		// Use cached version from Create() if available, otherwise fetch it
		if d.relayerVersion != "" {
			version = d.relayerVersion
		} else {
			// Fallback: fetch version if Start() is called without Create() first
			// Cache it so subsequent Start()/Restart() calls reuse the same version
			d.relayerVersion = GetRapidRelayerVersion()
			version = d.relayerVersion
		}
	}

	imageName, err := d.getImageName(version)
	if err != nil {
		return err
	}

	// Build binds (volume mounts)
	binds := []string{
		fmt.Sprintf("%s:/app/data", volumeName),
	}

	// Add additional volume mounts
	additionalMounts, err := d.getAdditionalVolumeMounts()
	if err != nil {
		return err
	}
	binds = append(binds, additionalMounts...)

	// Get port bindings
	portBindings, exposedPorts, err := d.getPortBindings()
	if err != nil {
		return err
	}

	// Get command arguments
	cmdArgs, err := d.getCommandArgs()
	if err != nil {
		return err
	}

	// Container configuration
	config := &container.Config{
		Image:        imageName,
		Cmd:          cmdArgs,
		ExposedPorts: exposedPorts,
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		Binds:        binds,
		PortBindings: portBindings,
		NetworkMode:  "host",
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Create container
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, serviceName)
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	return nil
}

func (d *Docker) Stop() error {
	// For Rollytics, use docker compose
	if d.commandName == Rollytics {
		return d.stopDockerCompose()
	}

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	serviceName, err := d.GetServiceName()
	if err != nil {
		return err
	}

	// Stop container
	timeout := 10 // seconds
	if err := cli.ContainerStop(ctx, serviceName, container.StopOptions{Timeout: &timeout}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to stop container: %v", err)
		}
	}

	// Remove the container after stopping
	if err := cli.ContainerRemove(ctx, serviceName, container.RemoveOptions{}); err != nil {
		if !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to remove container: %v", err)
		}
	}

	return nil
}

func (d *Docker) Log(n int) error {
	// For Rollytics, use docker compose
	if d.commandName == Rollytics {
		return d.logDockerCompose(n)
	}

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	serviceName, err := d.GetServiceName()
	if err != nil {
		return err
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       fmt.Sprintf("%d", n),
	}

	out, err := cli.ContainerLogs(ctx, serviceName, options)
	if err != nil {
		return err
	}
	defer out.Close()

	// Demultiplex stdout and stderr streams
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return err
}

func (d *Docker) getImageName(version string) (string, error) {
	baseImage := "ghcr.io/initia-labs"

	switch d.commandName {
	case Minitia:
		// TODO: add minitia image name
		panic("minitia image name is not supported")
		// return fmt.Sprintf("%s/minitiad:%s", baseImage, version), nil
	case UpgradableInitia, NonUpgradableInitia:
		return fmt.Sprintf("%s/initiad:%s", baseImage, version), nil
	case OPinitExecutor:
		return fmt.Sprintf("%s/opinit-bots:%s", baseImage, version), nil
	case OPinitChallenger:
		return fmt.Sprintf("%s/opinit-bots:%s", baseImage, version), nil
	case Relayer:
		return fmt.Sprintf("%s/rapid-relayer:%s", baseImage, version), nil
	case Rollytics:
		if version == "" {
			version = "latest"
		}
		return fmt.Sprintf("%s/rollytics:%s", baseImage, version), nil
	default:
		return "", fmt.Errorf("unsupported command: %v", d.commandName)
	}
}

func (d *Docker) getAdditionalVolumeMounts() ([]string, error) {
	// Get the user's home directory
	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	switch d.commandName {
	case Relayer:
		relayerPath := filepath.Join(userHome, common.RelayerDirectory)
		return []string{
			fmt.Sprintf("%s:/config", relayerPath),
			fmt.Sprintf("%s:/syncInfo", relayerPath),
		}, nil
	default:
		return []string{}, nil
	}
}

func (d *Docker) getPortBindings() (nat.PortMap, nat.PortSet, error) {
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	var ports []string
	switch d.commandName {
	case Minitia:
		ports = []string{"26656", "26657", "1317", "8545", "9090"}
	case UpgradableInitia, NonUpgradableInitia:
		ports = []string{"26656", "26657", "1317", "9090"}
	case OPinitChallenger:
		ports = []string{"3000"}
	case OPinitExecutor:
		ports = []string{"3001"}
	case Relayer:
		ports = []string{"7010", "7011"}
	case Rollytics:
		// TODO: revisit port
		ports = []string{"8080"}
	default:
		return portBindings, exposedPorts, nil
	}

	for _, port := range ports {
		containerPort, err := nat.NewPort("tcp", port)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port %s: %v", port, err)
		}

		exposedPorts[containerPort] = struct{}{}
		portBindings[containerPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: port,
			},
		}
	}

	return portBindings, exposedPorts, nil
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
	case Rollytics:
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
	// For Rollytics, return docker-compose.yml path
	if d.commandName == Rollytics {
		return d.getComposeFilePath()
	}
	// Not needed for Docker implementation
	return "", nil
}

func (d *Docker) GetServiceBinaryAndHome() (string, string, error) {
	volumeName, err := d.getVolumeName()
	if err != nil {
		return "", "", err
	}
	// Return empty binary (not applicable for Docker) and volume name as home identifier
	// The actual data path inside Docker is /app/data, but the volume is managed by Docker
	return "", volumeName, nil
}

func (d *Docker) PruneLogs() error {
	// For Rollytics, use docker compose
	if d.commandName == Rollytics {
		// Docker compose doesn't have a direct prune logs command
		return nil
	}

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	serviceName, err := d.GetServiceName()
	if err != nil {
		return err
	}

	// Inspect container to get log file path
	containerJSON, err := cli.ContainerInspect(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %v", err)
	}

	// Truncate the log file directly
	logPath := strings.TrimSpace(containerJSON.LogPath)
	if logPath != "" {
		if err := os.Truncate(logPath, 0); err != nil {
			// If truncate fails (e.g., permission denied), try restarting the container
			return d.Restart()
		}
	}

	return nil
}

func (d *Docker) Restart() error {
	// For Rollytics, use docker compose
	if d.commandName == Rollytics {
		return d.restartDockerCompose()
	}

	if err := d.Stop(); err != nil {
		return err
	}
	return d.Start()
}

// RemoveVolume removes the Docker volume associated with this service
// This is useful for complete cleanup and should be called separately from Stop
func (d *Docker) RemoveVolume() error {
	// For Rollytics, use docker compose
	if d.commandName == Rollytics {
		return d.removeDockerComposeVolumes()
	}

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	volumeName, err := d.getVolumeName()
	if err != nil {
		return err
	}

	if err := cli.VolumeRemove(ctx, volumeName, false); err != nil {
		return fmt.Errorf("failed to remove volume: %v", err)
	}

	return nil
}
