package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/service"
	"github.com/spf13/cobra"
)

func minitiaIndexerCommand() *cobra.Command {
	shortDescription := "Index the rollup data"
	indexerCmd := &cobra.Command{
		Use:   "indexer",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
	}
	indexerCmd.AddCommand(
		rollyticsStartCommand(),
		rollyticsStopCommand(),
		rollyticsRestartCommand(),
		rollyticsLogCommand(),
	)
	return indexerCmd
}

func rollyticsStartCommand() *cobra.Command {
	shortDescription := "Start the rollytics service"
	launchCmd := &cobra.Command{
		Use:   "start",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := service.NewService(service.Rollytics, "")
			if err != nil {
				return err
			}

			userHome, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %v", err)
			}
			rollyticsHome := filepath.Join(userHome, common.RollyticsDirectory)
			if err = s.Create("", rollyticsHome); err != nil {
				return fmt.Errorf("failed to create service: %v", err)
			}

			err = s.Start()
			if err != nil {
				return err
			}
			fmt.Println("Started rollytics service. You can see the logs with `weave rollytics log`")
			return nil

		},
	}

	return launchCmd
}

func rollyticsStopCommand() *cobra.Command {
	shortDescription := "Stop the rollytics service"
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
	}
	stopCmd.RunE = func(cmd *cobra.Command, args []string) error {
		s, err := service.NewService(service.Rollytics, "")
		if err != nil {
			return err
		}
		return s.Stop()
	}
	return stopCmd
}

func rollyticsRestartCommand() *cobra.Command {
	shortDescription := "Restart the rollytics service"
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
	}
	restartCmd.RunE = func(cmd *cobra.Command, args []string) error {
		s, err := service.NewService(service.Rollytics, "")
		if err != nil {
			return err
		}
		return s.Restart()
	}
	return restartCmd
}

func rollyticsLogCommand() *cobra.Command {
	shortDescription := "Stream the logs of the rollytics service"
	logCmd := &cobra.Command{
		Use:   "log",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
	}
	logCmd.RunE = func(cmd *cobra.Command, args []string) error {
		s, err := service.NewService(service.Rollytics, "")
		if err != nil {
			return err
		}
		return s.Log(100)
	}
	return logCmd
}
