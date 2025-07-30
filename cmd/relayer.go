package cmd

// import (
// 	"fmt"
// 	"os"
// 	"path/filepath"

// 	tea "github.com/charmbracelet/bubbletea"
// 	"github.com/spf13/cobra"

// 	"github.com/initia-labs/weave/analytics"
// 	"github.com/initia-labs/weave/common"
// 	"github.com/initia-labs/weave/config"
// 	weavecontext "github.com/initia-labs/weave/context"
// 	"github.com/initia-labs/weave/models"
// 	"github.com/initia-labs/weave/models/relayer"
// 	"github.com/initia-labs/weave/service"
// )

// func RelayerCommand() *cobra.Command {
// 	shortDescription := "Relayer subcommands"
// 	cmd := &cobra.Command{
// 		Use:                        "relayer",
// 		Short:                      shortDescription,
// 		Long:                       fmt.Sprintf("%s.\n\n%s", shortDescription, RelayerHelperText),
// 		DisableFlagParsing:         true,
// 		SuggestionsMinimumDistance: 2,
// 	}

// 	cmd.AddCommand(
// 		relayerInitCommand(),
// 		relayerStartCommand(),
// 		relayerStopCommand(),
// 		relayerRestartCommand(),
// 		relayerLogCommand(),
// 	)

// 	return cmd
// }

// func relayerInitCommand() *cobra.Command {
// 	shortDescription := "Initialize and configure your relayer for IBC"
// 	initCmd := &cobra.Command{
// 		Use:   "init",
// 		Short: shortDescription,
// 		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RelayerHelperText),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			analytics.TrackRunEvent(cmd, args, analytics.SetupRelayerFeature, analytics.NewEmptyEvent())
// 			ctx := weavecontext.NewAppContext(relayer.NewRelayerState())
// 			minitiaHome, _ := cmd.Flags().GetString(FlagMinitiaHome)
// 			ctx = weavecontext.SetMinitiaHome(ctx, minitiaHome)

// 			if config.IsFirstTimeSetup() {
// 				checkerCtx := weavecontext.NewAppContext(models.NewExistingCheckerState())
// 				if finalModel, err := tea.NewProgram(models.NewGasStationMethodSelect(checkerCtx), tea.WithAltScreen()).Run(); err != nil {
// 					return err
// 				} else {
// 					fmt.Println(finalModel.View())
// 					if _, ok := finalModel.(*models.WeaveAppSuccessfullyInitialized); !ok {
// 						return nil
// 					}
// 				}
// 			}

// 			model, err := relayer.NewRollupSelect(ctx)
// 			if err != nil {
// 				return err
// 			}
// 			if finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
// 				return err
// 			} else {
// 				fmt.Println(finalModel.View())
// 				return nil
// 			}
// 		},
// 	}

// 	homeDir, err := os.UserHomeDir()
// 	if err != nil {
// 		panic(fmt.Errorf("cannot get user home directory: %v", err))
// 	}

// 	initCmd.Flags().String(FlagMinitiaHome, filepath.Join(homeDir, common.MinitiaDirectory), "Rollup application directory to fetch artifacts from if existed")

// 	return initCmd
// }

// func relayerStartCommand() *cobra.Command {
// 	shortDescription := "Start the relayer service"
// 	startCmd := &cobra.Command{
// 		Use:     "start",
// 		Short:   shortDescription,
// 		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, RelayerHelperText),
// 		PreRunE: isInitiated(service.Relayer),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			detach, err := cmd.Flags().GetBool(FlagDetach)
// 			if err != nil {
// 				return err
// 			}

// 			updateClient, err := cmd.Flags().GetString(FlagUpdateClient)
// 			if err != nil {
// 				return err
// 			}

// 			switch updateClient {
// 			case "true":
// 				err = relayer.UpdateClientFromConfig()
// 				if err != nil {
// 					return err
// 				}
// 			case "false":
// 			default:
// 				return fmt.Errorf("invalid update-client flag value: %q, expected 'true' or 'false'", updateClient)
// 			}

// 			s, err := service.NewService(service.Relayer)
// 			if err != nil {
// 				return err
// 			}

// 			if detach {
// 				err = s.Start()
// 				if err != nil {
// 					return err
// 				}
// 				fmt.Println("Started relayer service. You can see the logs with `weave relayer log`")
// 				return nil
// 			}

// 			return service.NonDetachStart(s)
// 		},
// 	}
// 	startCmd.Flags().String(FlagUpdateClient, "true", "Update light clients with new header information before starting the relayer (can be 'true' or 'false')")
// 	startCmd.Flags().BoolP(FlagDetach, "d", false, "Run the relayer service in detached mode")

// 	return startCmd
// }

// func relayerStopCommand() *cobra.Command {
// 	shortDescription := "Stop the relayer service"
// 	stopCmd := &cobra.Command{
// 		Use:     "stop",
// 		Short:   shortDescription,
// 		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, RelayerHelperText),
// 		PreRunE: isInitiated(service.Relayer),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			s, err := service.NewService(service.Relayer)
// 			if err != nil {
// 				return err
// 			}
// 			err = s.Stop()
// 			if err != nil {
// 				return err
// 			}
// 			fmt.Println("Stopped relayer service.")
// 			return nil
// 		},
// 	}

// 	return stopCmd
// }

// func relayerRestartCommand() *cobra.Command {
// 	shortDescription := "Restart the relayer service"
// 	restartCmd := &cobra.Command{
// 		Use:     "restart",
// 		Short:   shortDescription,
// 		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, RelayerHelperText),
// 		PreRunE: isInitiated(service.Relayer),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			s, err := service.NewService(service.Relayer)
// 			if err != nil {
// 				return err
// 			}
// 			err = s.Restart()
// 			if err != nil {
// 				return err
// 			}

// 			fmt.Println("Started the relayer service. You can see the logs with `weave relayer log`")
// 			return nil
// 		},
// 	}

// 	return restartCmd
// }

// func relayerLogCommand() *cobra.Command {
// 	shortDescription := "Stream the logs of the relayer service"
// 	logCmd := &cobra.Command{
// 		Use:   "log",
// 		Short: shortDescription,
// 		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, RelayerHelperText),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			n, err := cmd.Flags().GetInt(FlagN)
// 			if err != nil {
// 				return err
// 			}

// 			s, err := service.NewService(service.Relayer)
// 			if err != nil {
// 				return err
// 			}
// 			return s.Log(n)
// 		},
// 	}

// 	logCmd.Flags().IntP(FlagN, FlagN, 100, "previous log lines to show")

// 	return logCmd
// }
