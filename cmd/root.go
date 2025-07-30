package cmd

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/config"
)

var Version string

func Execute() error {
	rootCmd := &cobra.Command{
		Use:  "weave",
		Long: WeaveHelperText,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			viper.AutomaticEnv()
			viper.SetEnvPrefix("weave")
			if err := config.InitializeConfig(); err != nil {
				return err
			}
			analytics.Initialize(Version)

			// Skip LZ4 check for certain commands that don't need it
			if cmd.Name() != "version" && cmd.Name() != "analytics" {
				if _, err := exec.LookPath("lz4"); err != nil {
					return fmt.Errorf("lz4 is not installed. Please install it first:\n" +
						"- For macOS: Run 'brew install lz4'\n" +
						"- For Ubuntu/Debian: Run 'apt-get install lz4'\n" +
						"- For other Linux distributions: Use your package manager to install lz4")
				}
			}

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			analytics.Client.Flush()
			analytics.Client.Shutdown()
			return nil
		},
	}

	rootCmd.AddCommand(
		InitCommand(),
		InitiaCommand(),
		GasStationCommand(),
		VersionCommand(),
		UpgradeCommand(),
		MinitiaCommand(),
		OPInitBotsCommand(),
		// RelayerCommand(),
		AnalyticsCommand(),
	)

	return rootCmd.ExecuteContext(context.Background())
}
