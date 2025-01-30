package cmd

import (
	"fmt"

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

			serviceBinary, serviceHome, err := service.GetServiceBinaryAndHome()
			if err != nil {
				return fmt.Errorf("could not determine %v binary and home directory: %w", prettyName, err)
			}

			if !weaveio.FileOrFolderExists(serviceHome) {
				return fmt.Errorf("home directory %s not found", serviceHome)
			}

			if !weaveio.FileOrFolderExists(serviceBinary) {
				return fmt.Errorf("%s binary not found at %s", prettyName, serviceBinary)
			}

			serviceFile, err := service.GetServiceFile()
			if err != nil {
				return fmt.Errorf("could not get service file for %s: %w", prettyName, err)
			}

			if !weaveio.FileOrFolderExists(serviceFile) {
				return fmt.Errorf("service file %s not found", serviceFile)
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
