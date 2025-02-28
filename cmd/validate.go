package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/initia-labs/weave/config"
	weaveio "github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/service"
)

func isInitiated(cmd service.Command) func(_ *cobra.Command, _ []string) error {
	return func(_ *cobra.Command, _ []string) error {
		if err := func() error {
			// Check if Docker is installed and running
			if err := exec.Command("docker", "info").Run(); err != nil {
				return fmt.Errorf("docker is not running or not installed: %w", err)
			}

			// Check if image is pulled or not
			imageURL := config.GetCommandImageURL(cmd.Name)
			inspectCmd := exec.Command("docker", "image", "inspect", imageURL)
			if err := inspectCmd.Run(); err != nil {
				return fmt.Errorf("required docker image %s not found", imageURL)
			}

			// Check if the home directory exists (for bind mount)
			serviceHome := config.GetCommandHome(cmd.Name)
			if !weaveio.FileOrFolderExists(serviceHome) {
				return fmt.Errorf("home directory %s not found", serviceHome)
			}

			return nil
		}(); err != nil {
			return fmt.Errorf("weave %s is not properly configured: %w: run `weave %s` to setup", cmd.Name, err, cmd.InitCommand)
		}

		return nil
	}
}
