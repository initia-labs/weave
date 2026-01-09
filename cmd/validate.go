package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/initia-labs/weave/common"
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
			svc, err := service.NewService(cmd, "")
			if err != nil {
				return fmt.Errorf("could not create %v service: %w", prettyName, err)
			}

			serviceFile, err := svc.GetServiceFile()
			if err != nil {
				return fmt.Errorf("could not get service file for %s: %w", prettyName, err)
			}

			if serviceFile != "" {
				if !weaveio.FileOrFolderExists(serviceFile) {
					return fmt.Errorf("service file %s not found", serviceFile)
				}

				serviceBinary, serviceHome, err := svc.GetServiceBinaryAndHome()
				if err != nil {
					return fmt.Errorf("could not determine %v binary and home directory: %w", prettyName, err)
				}

				if !weaveio.FileOrFolderExists(serviceHome) {
					return fmt.Errorf("home directory %s not found", serviceHome)
				}

				if !weaveio.FileOrFolderExists(serviceBinary) {
					return fmt.Errorf("%s binary not found at %s", prettyName, serviceBinary)
				}
			} else {
				// Validate Docker-backed services
				if cmd == service.Relayer {
					// Check if Docker CLI executable exists
					if _, err := exec.LookPath("docker"); err != nil {
						return fmt.Errorf("docker CLI executable not found: %w", err)
					}

					// Verify the relayer home directory exists
					userHome, err := os.UserHomeDir()
					if err != nil {
						return fmt.Errorf("could not get user home directory: %w", err)
					}

					relayerHome := filepath.Join(userHome, common.RelayerDirectory)
					if !weaveio.FileOrFolderExists(relayerHome) {
						return fmt.Errorf("relayer home directory %s not found", relayerHome)
					}

					// Confirm the presence of config.json in the relayer home
					configPath := filepath.Join(relayerHome, "config.json")
					if !weaveio.FileOrFolderExists(configPath) {
						return fmt.Errorf("config.json not found at %s", configPath)
					}
				}
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
