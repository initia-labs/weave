package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/config"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/crypto"
	"github.com/initia-labs/weave/models"
	"github.com/initia-labs/weave/registry"
)

func GasStationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "gas-station",
		Short:                      "Gas Station subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	cmd.AddCommand(
		gasStationSetupCommand(),
		gasStationShowCommand(),
	)

	return cmd
}

func gasStationSetupCommand() *cobra.Command {
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup Gas Station account on Initia and Celestia for funding the OPinit-bots or relayer to send transactions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			analytics.TrackRunEvent(cmd, args, analytics.SetupGasStationFeature, analytics.NewEmptyEvent())
			ctx := weavecontext.NewAppContext(models.NewExistingCheckerState())
			if finalModel, err := tea.NewProgram(models.NewGasStationMethodSelect(ctx), tea.WithAltScreen()).Run(); err != nil {
				return err
			} else {
				fmt.Println(finalModel.View())
				if _, ok := finalModel.(*models.WeaveAppSuccessfullyInitialized); !ok {
					return nil
				}
			}

			analytics.TrackCompletedEvent(analytics.SetupGasStationFeature)
			fmt.Println("Loading Gas Station balances...")
			return showGasStationBalance()
		},
	}

	return setupCmd
}

func getBalance(chainType registry.ChainType, address string) (*cosmosutils.Coins, error) {
	chainRegistry, err := registry.GetChainRegistry(chainType)
	if err != nil {
		return nil, fmt.Errorf("failed to load chainRegistry: %v", err)
	}

	baseUrl, err := chainRegistry.GetActiveLcd()
	if err != nil {
		return nil, fmt.Errorf("failed to get active lcd for %s: %v", chainType, err)
	}

	return cosmosutils.QueryBankBalances(baseUrl, address)
}

func getMaxWidth(coinGroups ...*cosmosutils.Coins) int {
	maxAmountWidth := 0
	maxDenomWidth := 0

	for _, coins := range coinGroups {
		for _, coin := range *coins {
			if len(coin.Amount) > maxAmountWidth {
				maxAmountWidth = len(coin.Amount)
			}
			if len(coin.Denom) > maxDenomWidth {
				maxDenomWidth = len(coin.Denom)
			}
		}
	}

	// Add 1 space here for the space between amount and denom
	return maxAmountWidth + maxDenomWidth + 1
}

func showGasStationBalance() error {
	gasStationMnemonic := config.GetGasStationMnemonic()
	initiaGasStationAddress, err := crypto.MnemonicToBech32Address("init", gasStationMnemonic)
	if err != nil {
		return err
	}
	celestiaGasStationAddress, err := crypto.MnemonicToBech32Address("celestia", gasStationMnemonic)
	if err != nil {
		return err
	}

	initiaL1TestnetBalances, err := getBalance(registry.InitiaL1Testnet, initiaGasStationAddress)
	if err != nil {
		return err
	}

	celestiaTestnetBalance, err := getBalance(registry.CelestiaTestnet, celestiaGasStationAddress)
	if err != nil {
		return err
	}

	celestiaMainnetBalance, err := getBalance(registry.CelestiaMainnet, celestiaGasStationAddress)
	if err != nil {
		return err
	}

	// Convert micro units to standard units
	convertedInitiaBalances := initiaL1TestnetBalances.ConvertMicroUnits(map[string]string{
		"uinit": "INIT",
	})
	convertedCelestiaTestnetBalance := celestiaTestnetBalance.ConvertMicroUnits(map[string]string{
		"utia": "TIA",
	})
	convertedCelestiaMainnetBalance := celestiaMainnetBalance.ConvertMicroUnits(map[string]string{
		"utia": "TIA",
	})

	maxWidth := getMaxWidth(convertedInitiaBalances, convertedCelestiaTestnetBalance, convertedCelestiaMainnetBalance)
	if maxWidth < len(cosmosutils.NoBalancesText) {
		maxWidth = len(cosmosutils.NoBalancesText)
	}
	fmt.Printf("\n⛽️ Initia Address: %s\n\nTestnet\n%s\n\n", initiaGasStationAddress, convertedInitiaBalances.Render(maxWidth))
	fmt.Printf("⛽️ Celestia Address: %s\n\nTestnet\n%s\nMainnet\n%s\n\n", celestiaGasStationAddress, convertedCelestiaTestnetBalance.Render(maxWidth), convertedCelestiaMainnetBalance.Render(maxWidth))
	fmt.Printf("💧 You can get testnet INIT from -> https://faucet.testnet.initia.xyz.\n💧 For testnet TIA, please refer to -> https://docs.celestia.org/how-to-guides/mocha-testnet#mocha-testnet-faucet\n")

	return nil
}

func gasStationShowCommand() *cobra.Command {
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show Initia and Celestia Gas Station addresses and balances",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.IsFirstTimeSetup() {
				fmt.Println("Please setup Gas Station first, by running `gas-station setup`")
				return nil
			}

			return showGasStationBalance()
		},
	}

	return showCmd
}
