package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/config"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/models"
	"github.com/initia-labs/weave/models/minitia"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/types"
)

type minitiaConfigKey struct{}

var (
	validVMOptions = []string{"evm", "move", "wasm"}
)

func MinitiaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "rollup",
		Short:                      "Rollup subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	cmd.AddCommand(
		minitiaLaunchCommand(),
		minitiaStartCommand(),
		minitiaStopCommand(),
		minitiaRestartCommand(),
		minitiaLogCommand(),
	)

	return cmd
}

func validateVMFlag(vmValue string) error {
	for _, option := range validVMOptions {
		if vmValue == option {
			return nil
		}
	}
	return fmt.Errorf("invalid value for --vm. Valid options are: %s", strings.Join(validVMOptions, ", "))
}

func loadAndParseMinitiaConfig(path string) (*types.MinitiaConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var minitiaConfig types.MinitiaConfig
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&minitiaConfig); err != nil {
		return nil, err
	}

	return &minitiaConfig, nil
}

func minitiaLaunchCommand() *cobra.Command {
	launchCmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch a new rollup from scratch",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString(FlagWithConfig)
			vm, _ := cmd.Flags().GetString(FlagVm)

			if configPath != "" && vm == "" {
				return fmt.Errorf("the --vm flag is required when using --with-config")
			}
			if configPath == "" && vm != "" {
				return fmt.Errorf("the --vm flag can only be used with --with-config")
			}

			if configPath != "" {
				minitiaConfig, err := loadAndParseMinitiaConfig(configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
				cmd.SetContext(context.WithValue(cmd.Context(), minitiaConfigKey{}, minitiaConfig))
			}

			if vm != "" {
				if err := validateVMFlag(vm); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			analytics.TrackRunEvent(cmd, analytics.RollupComponent)
			minitiaHome, err := cmd.Flags().GetString(FlagMinitiaHome)
			if err != nil {
				return err
			}

			state := minitia.NewLaunchState()

			configPath, _ := cmd.Flags().GetString(FlagWithConfig)
			if configPath != "" {
				minitiaConfig, ok := cmd.Context().Value(minitiaConfigKey{}).(*types.MinitiaConfig)
				if !ok {
					return fmt.Errorf("failed to retrieve configuration from context")
				}

				vm, _ := cmd.Flags().GetString(FlagVm)
				version, downloadURL, err := cosmosutils.GetLatestMinitiaVersion(vm)
				if err != nil {
					return err
				}

				state.PrepareLaunchingWithConfig(vm, version, downloadURL, configPath, minitiaConfig)
			}

			force, _ := cmd.Flags().GetBool(FlagForce)

			if force {
				if err = io.DeleteDirectory(minitiaHome); err != nil {
					return fmt.Errorf("failed to delete %s: %v", minitiaHome, err)
				}
			}

			ctx := weavecontext.NewAppContext(*state)
			ctx = weavecontext.SetMinitiaHome(ctx, minitiaHome)

			if config.IsFirstTimeSetup() {
				checkerCtx := weavecontext.NewAppContext(models.NewExistingCheckerState())
				checkerCtx = weavecontext.SetMinitiaHome(checkerCtx, minitiaHome)
				if finalModel, err := tea.NewProgram(models.NewGasStationMethodSelect(checkerCtx), tea.WithAltScreen()).Run(); err != nil {
					return err
				} else {
					fmt.Println(finalModel.View())
					if _, ok := finalModel.(*models.WeaveAppSuccessfullyInitialized); !ok {
						return nil
					}
				}
			}

			if finalModel, err := tea.NewProgram(minitia.NewExistingMinitiaChecker(ctx), tea.WithAltScreen()).Run(); err != nil {
				return err
			} else {
				fmt.Println(finalModel.View())
				analytics.TrackCompletedEvent(cmd, analytics.RollupComponent)
				return nil
			}
		},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("cannot get user home directory: %v", err))
	}

	launchCmd.Flags().String(FlagMinitiaHome, filepath.Join(homeDir, common.MinitiaDirectory), "The rollup application home directory")
	launchCmd.Flags().String(FlagWithConfig, "", "launch using an existing rollup config file. The argument should be the path to the config file.")
	launchCmd.Flags().String(FlagVm, "", fmt.Sprintf("vm to be used. Required when using --with-config. Valid options are: %s", strings.Join(validVMOptions, ", ")))
	launchCmd.Flags().BoolP(FlagForce, "f", false, "force the launch by deleting the existing .minitia directory if it exists.")

	return launchCmd
}

func minitiaStartCommand() *cobra.Command {
	launchCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the rollup full node application.",
		RunE: func(cmd *cobra.Command, args []string) error {
			detach, err := cmd.Flags().GetBool(FlagDetach)
			if err != nil {
				return err
			}

			s, err := service.NewService(service.Minitia)
			if err != nil {
				return err
			}

			if detach {
				err = s.Start()
				if err != nil {
					return err
				}
				fmt.Println("Started rollup full node application. You can see the logs with `weave rollup log`")
				return nil
			}

			return service.NonDetachStart(s)
		},
	}

	launchCmd.Flags().BoolP(FlagDetach, "d", false, "Run the rollup full node application in detached mode")

	return launchCmd
}

func minitiaStopCommand() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the rollup full node application.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := service.NewService(service.Minitia)
			if err != nil {
				return err
			}
			err = s.Stop()
			if err != nil {
				return err
			}
			fmt.Println("Stopped the rollup full node application.")
			return nil
		},
	}

	return startCmd
}

func minitiaRestartCommand() *cobra.Command {
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the rollup full node application.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := service.NewService(service.Minitia)
			if err != nil {
				return err
			}
			err = s.Restart()
			if err != nil {
				return err
			}

			fmt.Println("Restart rollup full node application. You can see the logs with `weave rollup log`")
			return nil
		},
	}

	return restartCmd
}

func minitiaLogCommand() *cobra.Command {
	logCmd := &cobra.Command{
		Use:   "log",
		Short: "Stream the logs of the rollup full node application.",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := cmd.Flags().GetInt(FlagN)
			if err != nil {
				return err
			}

			s, err := service.NewService(service.Minitia)
			if err != nil {
				return err
			}
			return s.Log(n)
		},
	}

	logCmd.Flags().IntP(FlagN, FlagN, 100, "previous log lines to show")

	return logCmd
}
