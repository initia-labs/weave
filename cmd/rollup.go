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
	shortDescription := "Rollup subcommands"
	cmd := &cobra.Command{
		Use:                        "rollup",
		Short:                      shortDescription,
		Long:                       fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
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
	shortDescription := "Launch a new rollup from scratch"
	launchCmd := &cobra.Command{
		Use:   "launch",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
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
			minitiaHome, err := cmd.Flags().GetString(FlagMinitiaHome)
			if err != nil {
				return err
			}
			opinitHome, err := cmd.Flags().GetString(FlagOPInitHome)
			if err != nil {
				return err
			}
			configPath, _ := cmd.Flags().GetString(FlagWithConfig)
			vm, _ := cmd.Flags().GetString(FlagVm)
			force, _ := cmd.Flags().GetBool(FlagForce)
			state := minitia.NewLaunchState()
			events := analytics.NewEmptyEvent()
			if configPath != "" {
				events.Add(analytics.WithConfigKey, true).
					Add(analytics.VmKey, vm)
			}
			analytics.TrackRunEvent(cmd, args, analytics.RollupLaunchFeature, events)
			if configPath != "" {
				if io.FileOrFolderExists(minitiaHome) && !force {
					return fmt.Errorf("existing %s folder detected. Use --force or -f to override", minitiaHome)
				}

				minitiaConfig, ok := cmd.Context().Value(minitiaConfigKey{}).(*types.MinitiaConfig)
				if !ok {
					return fmt.Errorf("failed to retrieve configuration from context")
				}

				version, downloadURL, err := cosmosutils.GetLatestMinitiaVersion(vm)
				if err != nil {
					return err
				}

				state.PrepareLaunchingWithConfig(vm, version, downloadURL, configPath, minitiaConfig)
			}

			if force {
				if err = io.DeleteDirectory(minitiaHome); err != nil {
					return fmt.Errorf("failed to delete %s: %v", minitiaHome, err)
				}
			}

			ctx := weavecontext.NewAppContext(*state)
			ctx = weavecontext.SetMinitiaHome(ctx, minitiaHome)
			ctx = weavecontext.SetOPInitHome(ctx, opinitHome)
			if config.IsFirstTimeSetup() {
				checkerCtx := weavecontext.NewAppContext(models.NewExistingCheckerState())
				checkerCtx = weavecontext.SetMinitiaHome(checkerCtx, minitiaHome)
				checkerCtx = weavecontext.SetOPInitHome(checkerCtx, opinitHome)
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
				return nil
			}
		},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("cannot get user home directory: %v", err))
	}

	launchCmd.Flags().String(FlagMinitiaHome, filepath.Join(homeDir, common.MinitiaDirectory), "The rollup application home directory")
	launchCmd.Flags().String(FlagOPInitHome, filepath.Join(homeDir, common.OPinitDirectory), "The OPinit application home directory")
	launchCmd.Flags().String(FlagWithConfig, "", "Launch using an existing rollup config file. The argument should be the path to the config file")
	launchCmd.Flags().String(FlagVm, "", fmt.Sprintf("VM to be used. Required when using --with-config. Valid options are: %s", strings.Join(validVMOptions, ", ")))
	launchCmd.Flags().BoolP(FlagForce, "f", false, "Force the launch by deleting the existing .minitia directory if it exists")

	return launchCmd
}

func minitiaStartCommand() *cobra.Command {
	shortDescription := "Start the rollup full node service"
	launchCmd := &cobra.Command{
		Use:     "start",
		Short:   shortDescription,
		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
		PreRunE: isInitiated(service.Minitia),
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
				fmt.Println("Started rollup full node service. You can see the logs with `weave rollup log`")
				return nil
			}

			return service.NonDetachStart(s)
		},
	}

	launchCmd.Flags().BoolP(FlagDetach, "d", false, "Run the rollup full node service in detached mode")

	return launchCmd
}

func minitiaStopCommand() *cobra.Command {
	shortDescription := "Stop the rollup full node service"
	startCmd := &cobra.Command{
		Use:     "stop",
		Short:   shortDescription,
		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
		PreRunE: isInitiated(service.Minitia),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := service.NewService(service.Minitia)
			if err != nil {
				return err
			}
			err = s.Stop()
			if err != nil {
				return err
			}
			fmt.Println("Stopped the rollup full node service.")
			return nil
		},
	}

	return startCmd
}

func minitiaRestartCommand() *cobra.Command {
	shortDescription := "Restart the rollup full node service"
	restartCmd := &cobra.Command{
		Use:     "restart",
		Short:   shortDescription,
		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
		PreRunE: isInitiated(service.Minitia),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := service.NewService(service.Minitia)
			if err != nil {
				return err
			}
			err = s.Restart()
			if err != nil {
				return err
			}

			fmt.Println("Restart rollup full node service. You can see the logs with `weave rollup log`")
			return nil
		},
	}

	return restartCmd
}

func minitiaLogCommand() *cobra.Command {
	shortDescription := "Stream the logs of the rollup full node service"
	logCmd := &cobra.Command{
		Use:   "log",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RollupHelperText),
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
