package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/initia"
	"github.com/initia-labs/weave/service"
)

func InitiaCommand() *cobra.Command {
	shortDescription := "Initia node subcommands"
	cmd := &cobra.Command{
		Use:                        "initia",
		Short:                      shortDescription,
		Long:                       fmt.Sprintf("%s.\n\n%s", shortDescription, L1NodeHelperText),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	cmd.AddCommand(
		initiaInitCommand(),
		initiaStartCommand(),
		initiaStopCommand(),
		initiaRestartCommand(),
		initiaLogCommand(),
	)

	return cmd
}

func initiaInitCommand() *cobra.Command {
	shortDescription := "Bootstrap your Initia full node"
	initCmd := &cobra.Command{
		Use:   "init",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, L1NodeHelperText),
		RunE: func(cmd *cobra.Command, args []string) error {
			analytics.TrackRunEvent(cmd, args, analytics.SetupL1NodeFeature, analytics.NewEmptyEvent())
			initiaHome, err := cmd.Flags().GetString(FlagInitiaHome)
			if err != nil {
				return err
			}

			ctx := weavecontext.NewAppContext(initia.NewRunL1NodeState())
			ctx = weavecontext.SetInitiaHome(ctx, initiaHome)
			model, err := initia.NewRunL1NodeNetworkSelect(ctx)
			if err != nil {
				return err
			}

			if finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
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

	initCmd.Flags().String(FlagInitiaHome, filepath.Join(homeDir, common.InitiaDirectory), "The Initia application home directory")

	return initCmd
}

func initiaStartCommand() *cobra.Command {
	shortDescription := "Start Initia full node service"
	startCmd := &cobra.Command{
		Use:     "start",
		Short:   shortDescription,
		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, L1NodeHelperText),
		PreRunE: isInitiated(service.UpgradableInitia),
		RunE: func(cmd *cobra.Command, args []string) error {
			detach, err := cmd.Flags().GetBool(FlagDetach)
			if err != nil {
				return err
			}

			s, err := service.NewService(service.UpgradableInitia)
			if err != nil {
				return err
			}

			if detach {
				err = s.Start()
				if err != nil {
					return err
				}
				fmt.Println("Started Initia full node service. You can see the logs with `weave initia log`")
				return nil
			}

			return service.NonDetachStart(s)
		},
	}

	startCmd.Flags().BoolP(FlagDetach, "d", false, "Run the initiad full node service in detached mode")

	return startCmd
}

func initiaStopCommand() *cobra.Command {
	shortDescription := "Stop Initia full node service"
	stopCmd := &cobra.Command{
		Use:     "stop",
		Short:   shortDescription,
		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, L1NodeHelperText),
		PreRunE: isInitiated(service.UpgradableInitia),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := service.NewService(service.UpgradableInitia)
			if err != nil {
				return err
			}
			err = s.Stop()
			if err != nil {
				return err
			}
			fmt.Println("Stopped Initia full node service.")
			return nil
		},
	}

	return stopCmd
}

func initiaRestartCommand() *cobra.Command {
	shortDescription := "Restart Initia full node service"
	restartCmd := &cobra.Command{
		Use:     "restart",
		Short:   shortDescription,
		Long:    fmt.Sprintf("%s.\n\n%s", shortDescription, L1NodeHelperText),
		PreRunE: isInitiated(service.UpgradableInitia),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := service.NewService(service.UpgradableInitia)
			if err != nil {
				return err
			}
			err = s.Restart()
			if err != nil {
				return err
			}

			fmt.Println("Started Initia full node service. You can see the logs with `weave initia log`")
			return nil
		},
	}

	return restartCmd
}

func initiaLogCommand() *cobra.Command {
	shortDescription := "Stream the logs of the Initia full node service"
	logCmd := &cobra.Command{
		Use:   "log",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, L1NodeHelperText),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := cmd.Flags().GetInt(FlagN)
			if err != nil {
				return err
			}

			s, err := service.NewService(service.UpgradableInitia)
			if err != nil {
				return err
			}
			return s.Log(n)
		},
	}

	logCmd.Flags().IntP(FlagN, FlagN, 100, "previous log lines to show")

	return logCmd
}
