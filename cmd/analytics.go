package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/config"
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
			analytics.TrackRunEvent(cmd, args, analytics.OptOutAnalyticsFeature, analytics.NewEmptyEvent())

			err := config.SetAnalyticsOptOut(true)
			if err != nil {
				return err
			}

			fmt.Println("Analytics disabled")
			analytics.TrackCompletedEvent(analytics.OptOutAnalyticsFeature)
			return nil
		},
	}

	return disableCmd
}
