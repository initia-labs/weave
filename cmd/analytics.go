package cmd

import (
	"fmt"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/config"
	"github.com/spf13/cobra"
)

func AnalyticsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "analytics",
		Short:                      "Configure analytics ex. enable/disable data collection",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	cmd.AddCommand(
		AnalyticsEnableCommand(),
		AnalyticsDisableCommand(),
	)

	return cmd
}

func AnalyticsEnableCommand() *cobra.Command {
	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Allow Weave to collect analytics data",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := config.SetAnalyticsOptOut(false)
			if err != nil {
				return err
			}

			// Initialize the analytics client so the event is tracked
			analytics.Initialize(Version)

			// Run after setting the config so the event is tracked
			analytics.TrackRunEvent(cmd, analytics.AnalyticsComponent)
			fmt.Println("Analytics enabled")
			return nil
		},
	}

	return enableCmd
}

func AnalyticsDisableCommand() *cobra.Command {
	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Do not allow Weave to collect analytics data",
		RunE: func(cmd *cobra.Command, args []string) error {
			analytics.TrackRunEvent(cmd, analytics.AnalyticsComponent)

			err := config.SetAnalyticsOptOut(true)
			if err != nil {
				return err
			}

			fmt.Println("Analytics disabled")
			return nil
		},
	}

	return disableCmd
}
