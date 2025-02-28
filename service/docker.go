package service

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/initia-labs/weave/config"
)

type Docker struct {
	command Command
}

func NewDocker(command Command) *Docker {
	return &Docker{
		command: command,
	}
}

func (d *Docker) Create(appHome string, customDockerImage string) error {
	// Create Docker network if it doesn't exist
	networkCmd := exec.Command("docker", "network", "create", "weave-network")
	_ = networkCmd.Run() // Ignore error if network already exists

	// Pull the appropriate image based on command type and version
	imageName := customDockerImage
	if imageName == "" {
		imageName = d.command.DefaultImageURL
	}

	err := config.SetCommandImageURL(d.command.Name, imageName)
	if err != nil {
		return fmt.Errorf("failed to set command image URL: %v", err)
	}

	err = config.SetCommandHome(d.command.Name, appHome)
	if err != nil {
		return fmt.Errorf("failed to set command home: %v", err)
	}

	pullCmd := exec.Command("docker", "pull", imageName)
	if output, err := pullCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull docker image: %v, output: %s", err, string(output))
	}

	return nil
}

func (d *Docker) buildArgs(detach bool, options []string, args []string) []string {
	imageName := config.GetCommandImageURL(d.command.Name)
	appHome := config.GetCommandHome(d.command.Name)

	dockerArgs := []string{"run"}

	if detach {
		dockerArgs = append(dockerArgs,
			"-d",
			"--name", d.command.Name,
			"--restart", "unless-stopped",
		)
	} else {
		dockerArgs = append(dockerArgs, "--rm")
	}

	// Common arguments for both Start and Run
	dockerArgs = append(dockerArgs,
		"--network", "weave-network",
		"-v", fmt.Sprintf("%s:/app/data", appHome),
	)

	dockerArgs = append(dockerArgs, options...)

	// Add the image name
	dockerArgs = append(dockerArgs, imageName)

	// Add the start command
	dockerArgs = append(dockerArgs, args...)

	return dockerArgs
}

func (d *Docker) Start(detach bool) error {
	args := d.buildArgs(detach, append(d.command.StartPortArgs, d.command.StartEnvArgs...), d.command.StartCommandArgs)
	cmd := exec.Command("docker", args...)

	if detach {
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start container: %v, output: %s", err, string(output))
		}

		return nil
	}

	// For attached mode, stream output directly
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d *Docker) Stop() error {
	cmd := exec.Command("docker", "stop", d.command.Name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop container: %v, output: %s", err, string(output))
	}

	// Remove the container after stopping
	rmCmd := exec.Command("docker", "rm", d.command.Name)
	_ = rmCmd.Run() // Ignore error if container doesn't exist

	return nil
}

func (d *Docker) Log(n int) error {
	cmd := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", n), "-f", d.command.Name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d *Docker) PruneLogs() error {
	cmd := exec.Command("docker", "logs", "--truncate", "0", d.command.Name)
	return cmd.Run()
}

func (d *Docker) Restart() error {
	if err := d.Stop(); err != nil {
		return err
	}
	return d.Start(true)
}

func (d *Docker) RunCmd(options []string, args ...string) *exec.Cmd {
	dockerArgs := d.buildArgs(false, options, args)
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}
