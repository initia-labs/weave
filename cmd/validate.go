package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	weaveio "github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/service"
)

func isInitiated(cmd service.CommandName) func(_ *cobra.Command, _ []string) error {
	return func(_ *cobra.Command, _ []string) error {
		prettyName, prettyErr := cmd.GetPrettyName()
		if prettyErr != nil {
			return fmt.Errorf("could not get pretty name: %w", prettyErr)
		}

		if err := func() error {
			service, err := service.NewService(cmd)
			if err != nil {
				return fmt.Errorf("could not create %v service: %w", prettyName, err)
			}

			// Check if Docker is installed and running
			if err := exec.Command("docker", "info").Run(); err != nil {
				return fmt.Errorf("docker is not running or not installed: %w", err)
			}

			imageName, err := service.getImageName("")
			if err != nil {
				return fmt.Errorf("could not get image name: %w", err)
			}

			inspectCmd := exec.Command("docker", "image", "inspect", imageName)
			if err := inspectCmd.Run(); err != nil {
				return fmt.Errorf("required docker image %s not found", imageName)
			}

			// Check if the home directory exists (for bind mount)
			_, serviceHome, err := service.GetServiceBinaryAndHome()
			if err != nil {
				return fmt.Errorf("could not determine %v home directory: %w", prettyName, err)
			}

			if !weaveio.FileOrFolderExists(serviceHome) {
				return fmt.Errorf("home directory %s not found", serviceHome)
			}

			return nil
		}(); err != nil {
			initCmd, initErr := cmd.GetInitCommand()
			if initErr != nil {
				return fmt.Errorf("could not get init command: %w", initErr)
			}

			return fmt.Errorf("weave %s is not properly configured: %w: run `weave %s` to setup", prettyName, err, initCmd)
		}

		return nil
	}
}
