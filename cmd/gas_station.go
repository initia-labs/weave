package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// Constants for denomination handling
const (
	utiaDenom    = "utia"
	tiaDenom     = "TIA"
	tiaExponent  = 6
	uinitDenom   = "uinit"
	initDenom    = "INIT"
	initExponent = 6

	testnetRegistryURL = "https://registry.testnet.initia.xyz/initia/assetlist.json"
	mainnetRegistryURL = "https://registry.initia.xyz/initia/assetlist.json"
)

type DenomUnit struct {
	Denom    string `json:"denom"`
	Exponent int    `json:"exponent"`
}

const (
	DefaultTimeout = 3 * time.Second
)

type Asset struct {
	DenomUnits []DenomUnit `json:"denom_units"`
	Base       string      `json:"base"`
	Display    string      `json:"display"`
}

type AssetList struct {
	Assets []Asset `json:"assets"`
}

// formatAmount formats an amount string with the given exponent
func formatAmount(amount string, exponent int) string {
	if len(amount) > exponent {
		decimalPos := len(amount) - exponent
		amount = amount[:decimalPos] + "." + amount[decimalPos:]
	} else {
		zeros := strings.Repeat("0", exponent-len(amount))
		amount = "0." + zeros + amount
	}

	// Trim trailing zeros and decimal point if necessary
	amount = strings.TrimRight(strings.TrimRight(amount, "0"), ".")
	if amount == "" {
		amount = "0"
	}
	return amount
}

func fetchInitiaRegistryAssetList(chainType registry.ChainType) (*AssetList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	registryURL := testnetRegistryURL
	if chainType == registry.InitiaL1Mainnet {
		registryURL = mainnetRegistryURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", registryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch asset list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var assetList AssetList
	if err := json.NewDecoder(resp.Body).Decode(&assetList); err != nil {
		return nil, fmt.Errorf("failed to decode asset list: %w", err)
	}
	return &assetList, nil
}

func convertToDisplayDenom(coins *cosmosutils.Coins, assetList *AssetList) *cosmosutils.Coins {
	if coins == nil {
		return nil
	}

	result := make(cosmosutils.Coins, 0, len(*coins))
	for _, coin := range *coins {
		displayCoin := coin

		// Handle special cases first
		switch coin.Denom {
		case utiaDenom:
			displayCoin.Amount = formatAmount(coin.Amount, tiaExponent)
			displayCoin.Denom = tiaDenom
			result = append(result, displayCoin)
			continue
		case uinitDenom:
			displayCoin.Amount = formatAmount(coin.Amount, initExponent)
			displayCoin.Denom = initDenom
			result = append(result, displayCoin)
			continue
		}

		// Handle other denoms via asset list
		if assetList != nil {
			if displayDenom, exponent := findHighestExponentDenom(assetList, coin.Denom); exponent > 0 {
				displayCoin.Amount = formatAmount(coin.Amount, exponent)
				displayCoin.Denom = displayDenom
			}
		}
		result = append(result, displayCoin)
	}
	return &result
}

// findHighestExponentDenom finds the denomination with the highest exponent for a given base denom
func findHighestExponentDenom(assetList *AssetList, baseDenom string) (string, int) {
	for _, asset := range assetList.Assets {
		if asset.Base == baseDenom {
			maxExponent := 0
			displayDenom := baseDenom
			for _, unit := range asset.DenomUnits {
				if unit.Exponent > maxExponent {
					maxExponent = unit.Exponent
					displayDenom = unit.Denom
				}
			}
			return displayDenom, maxExponent
		}
	}
	return baseDenom, 0
}

func GasStationCommand() *cobra.Command {
	shortDescription := "Gas Station subcommands"
	cmd := &cobra.Command{
		Use:                        "gas-station",
		Short:                      shortDescription,
		Long:                       fmt.Sprintf("%s.\n\n%s", shortDescription, GasStationHelperText),
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
	shortDescription := "Setup Gas Station account on Initia and Celestia for funding the OPinit-bots or relayer to send transactions"
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, GasStationHelperText),
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

	// Fetch asset list and convert denoms before printing
	testnetAssetList, err := fetchInitiaRegistryAssetList(registry.InitiaL1Testnet)
	if err != nil {
		// Log the error but continue with nil assetList
		fmt.Printf("Warning: Failed to fetch asset list: %v. Displaying original denominations.\n", err)
		testnetAssetList = nil
	}

	// Convert balances to display denoms
	initiaL1TestnetBalances = convertToDisplayDenom(initiaL1TestnetBalances, testnetAssetList)
	celestiaTestnetBalance = convertToDisplayDenom(celestiaTestnetBalance, nil) // nil assetList for Celestia
	celestiaMainnetBalance = convertToDisplayDenom(celestiaMainnetBalance, nil) // nil assetList for Celestia

	maxWidth := getMaxWidth(initiaL1TestnetBalances, celestiaTestnetBalance, celestiaMainnetBalance)
	if maxWidth < len(cosmosutils.NoBalancesText) {
		maxWidth = len(cosmosutils.NoBalancesText)
	}
	fmt.Printf("\n⛽️ Initia Address: %s\n\nTestnet\n%s\n\n", initiaGasStationAddress, initiaL1TestnetBalances.Render(maxWidth))
	fmt.Printf("⛽️ Celestia Address: %s\n\nTestnet\n%s\nMainnet\n%s\n\n", celestiaGasStationAddress, celestiaTestnetBalance.Render(maxWidth), celestiaMainnetBalance.Render(maxWidth))
	fmt.Printf("💧 You can get testnet INIT from -> https://faucet.testnet.initia.xyz.\n💧 For testnet TIA, please refer to -> https://docs.celestia.org/how-to-guides/mocha-testnet#mocha-testnet-faucet\n")

	return nil
}

func gasStationShowCommand() *cobra.Command {
	shortDescription := "Show Initia and Celestia Gas Station addresses and balances"
	showCmd := &cobra.Command{
		Use:   "show",
		Short: shortDescription,
		Long:  fmt.Sprintf("%s.\n\n%s", shortDescription, GasStationHelperText),
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.IsFirstTimeSetup() {
				fmt.Println("Please setup Gas Station first, by running `weave gas-station setup`")
				return nil
			}

			return showGasStationBalance()
		},
	}

	return showCmd
}
