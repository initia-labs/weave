package minitia_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/minitia"
	"github.com/initia-labs/weave/tests/integration"
)

const (
	TestMinitiaHome = ".minitia.weave.test"
)

func TestMinitiaLaunchWithExisting(t *testing.T) {
	t.Skip("Skipping minitia launch with existing test")
	_ = integration.SetupGasStation(t)

	ctx := context.NewAppContext(*minitia.NewLaunchState())
	ctx = context.SetMinitiaHome(ctx, TestMinitiaHome)

	firstModel := minitia.NewExistingMinitiaChecker(ctx)

	// Ensure that there is no previous Minitia home
	_, err := os.Stat(TestMinitiaHome)
	assert.NotNil(t, err)

	steps := integration.Steps{
		integration.WaitFetching,                   // wait checking for the existing rollup app
		integration.PressEnter,                     // press enter to select Testnet
		integration.PressEnter,                     // press enter to select Move vm for the rollup
		integration.WaitFetching,                   // wait for the fetching of the latest Move rollup version
		integration.TypeText("rollup-1"),           // type in the rollup chain id
		integration.PressEnter,                     // press enter to confirm the rollup chain id
		integration.PressTab,                       // press tab to use the default gas denom
		integration.PressEnter,                     // press enter to confirm using the default gas denom
		integration.PressTab,                       // press tab to use the default moniker
		integration.PressEnter,                     // press enter to confirm using the default moniker
		integration.PressTab,                       // press tab to use the default submission interval
		integration.PressEnter,                     // press enter to confirm using the default submission interval
		integration.PressTab,                       // press tab to use the default output submission period
		integration.PressEnter,                     // press enter to confirm using the default output submission period
		integration.PressDown,                      // press down once to select Initia L1 as the da layer
		integration.PressEnter,                     // press enter to confirm using Initia L1 as the da layer
		integration.PressEnter,                     // press enter to confirm enabling the oracle
		integration.PressEnter,                     // press enter to generate keys for system keys
		integration.WaitFetching,                   // wait for the fetching of gas station
		integration.PressDown,                      // press down once to select filling each account balance manually
		integration.PressEnter,                     // press enter to confirm selecting manual
		integration.TypeText("1234567"),            // type in the balance for the bridge executor
		integration.PressEnter,                     // press enter to confirm the balance for the bridge executor
		integration.TypeText("1000000"),            // type in the balance for the output submitter
		integration.PressEnter,                     // press enter to confirm the balance for the output submitter
		integration.TypeText("1111111"),            // type in the balance for the batch submitter
		integration.PressEnter,                     // press enter to confirm the balance for the batch submitter
		integration.TypeText("234567"),             // type in the balance for the challenger
		integration.PressEnter,                     // press enter to confirm the balance for the challenger
		integration.TypeText("123456789123456789"), // type in the balance for the L2 operator
		integration.PressEnter,                     // press enter to confirm the balance for the L2 operator
		integration.TypeText("123456789123456789"), // type in the balance for the L2 bridge executor
		integration.PressEnter,                     // press enter to confirm the balance for the L2 bridge executor
		integration.PressDown,                      // press down to select do not add more genesis account
		integration.PressEnter,                     // press enter to confirm the selection
		integration.WaitFetching,                   // wait for the downloading of the Move rollup binary
		integration.WaitFetching,                   // wait for the key generation
		integration.TypeText("continue"),           // type in to continue the process
		integration.PressEnter,                     // press enter to confirm continuing
		integration.TypeText("y"),                  // type in y to confirm the broadcast of transactions
		integration.PressEnter,                     // press enter to confirm transactions broadcast
	}

	finalModel := integration.RunProgramWithSteps(t, firstModel, steps)

	// Check the final state here
	assert.IsType(t, &minitia.TerminalState{}, finalModel)

	if _, ok := finalModel.(*minitia.TerminalState); ok {
		assert.True(t, ok)
	}

	// Check if Initia home has been created
	_, err = os.Stat(TestMinitiaHome)
	assert.Nil(t, err)

	configTomlPath := filepath.Join(TestMinitiaHome, "config/config.toml")
	appTomlPath := filepath.Join(TestMinitiaHome, "config/app.toml")

	if _, err := os.Stat(configTomlPath); os.IsNotExist(err) {
		t.Fatalf("Config toml file does not exist: %s", configTomlPath)
	}

	if _, err := os.Stat(appTomlPath); os.IsNotExist(err) {
		t.Fatalf("App toml file does not exist: %s", appTomlPath)
	}

	// Assert values
	integration.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
	integration.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
	integration.CompareTomlValue(t, configTomlPath, "statesync.enable", false)

	integration.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0umin")
	integration.CompareTomlValue(t, appTomlPath, "api.enable", true)

	artifactsConfigPath := filepath.Join(TestMinitiaHome, "artifacts/config.json")

	if _, err := os.Stat(artifactsConfigPath); os.IsNotExist(err) {
		t.Fatalf("Artifacts config json file does not exist: %s", artifactsConfigPath)
	}

	integration.CompareJsonValue(t, artifactsConfigPath, "l2_config.chain_id", "rollup-1")
	integration.CompareJsonValue(t, artifactsConfigPath, "l2_config.denom", "umin")
	integration.CompareJsonValue(t, artifactsConfigPath, "l2_config.moniker", "operator")
	integration.CompareJsonValue(t, artifactsConfigPath, "op_bridge.enable_oracle", true)

	// Try again with existing Minitia home
	ctx = context.NewAppContext(*minitia.NewLaunchState())
	ctx = context.SetMinitiaHome(ctx, TestMinitiaHome)

	firstModel = minitia.NewExistingMinitiaChecker(ctx)

	// Ensure that there is an existing Minitia home
	_, err = os.Stat(TestMinitiaHome)
	assert.Nil(t, err)

	steps = integration.Steps{
		integration.WaitFetching,         // wait checking for the existing rollup app
		integration.TypeText("delete"),   // type in delete to confirm removing the existing rollup app
		integration.PressEnter,           // press enter to confirm removing the rollup app
		integration.PressEnter,           // press enter to select Testnet
		integration.PressDown,            // press down once to select the Wasm rollup
		integration.PressEnter,           // press enter to confirm selecting the Wasm rollup
		integration.WaitFetching,         // wait for the fetching of the latest Move rollup version
		integration.TypeText("rollup-2"), // type in the rollup chain id
		integration.PressEnter,           // press enter to confirm the rollup chain id
		integration.TypeText("uroll"),    // type in the gas denom
		integration.PressEnter,           // press enter to confirm the gas denom
		integration.TypeText("computer"), // type in the moniker
		integration.PressEnter,           // press enter to confirm the moniker
		integration.PressTab,             // press tab to use the default submission interval
		integration.PressEnter,           // press enter to confirm using the default submission interval
		integration.PressTab,             // press tab to use the default output submission period
		integration.PressEnter,           // press enter to confirm using the default output submission period
		integration.PressDown,            // press down once to select Initia L1 as the da layer
		integration.PressEnter,           // press enter to confirm using Initia L1 as the da layer
		integration.PressEnter,           // press enter to confirm enabling the oracle
		integration.PressDown,            // press down once to select importing existing keys
		integration.PressEnter,           // press enter to confirm importing existing keys
		integration.TypeText("lonely fly lend protect mix order legal organ fruit donkey dog state"), // type in the mnemonic for the operator
		integration.PressEnter, // press enter to confirm the mnemonic
		integration.TypeText("boy salmon resist afford dog cereal first myth require enough sunny cargo"), // type in the mnemonic for the bridge executor
		integration.PressEnter, // press enter to confirm the mnemonic
		integration.TypeText("young diagram garment finish barrel output pledge borrow tonight frozen clerk sadness"), // type in the mnemonic for the output submitter
		integration.PressEnter, // press enter to confirm the mnemonic
		integration.TypeText("patrol search opera diary hidden giggle crisp together toy print lemon very"), // type in the mnemonic for the batch submitter
		integration.PressEnter, // press enter to confirm the mnemonic
		integration.TypeText("door radar exhibit equip mom beach drift harbor tomorrow tree long stereo"), // type in the mnemonic for the challenger
		integration.PressEnter,   // press enter to confirm the mnemonic
		integration.WaitFetching, // wait for the fetching of gas station
		integration.PressEnter,   // press enter to confirm using the default funding preset
		integration.PressEnter,   // press enter to confirm adding a genesis account
		integration.TypeText("init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr"), // type in the genesis account address
		integration.PressEnter,             // press enter to confirm the address
		integration.TypeText("1234567890"), // type in the genesis account initial balance
		integration.PressEnter,             // press enter to confirm the balance
		integration.PressDown,              // press down once to select not adding more accounts
		integration.PressEnter,             // press enter to confirm the selection
		integration.WaitFetching,           // wait for the downloading of the Wasm rollup binary
		integration.WaitFetching,           // wait for the key recovery
		integration.TypeText("y"),          // type in y to confirm the broadcast of transactions
		integration.PressEnter,             // press enter to confirm transactions broadcast
	}

	finalModel = integration.RunProgramWithSteps(t, firstModel, steps)
	defer integration.ClearTestDir(TestMinitiaHome)

	// Assert values
	integration.CompareJsonValue(t, artifactsConfigPath, "l2_config.chain_id", "rollup-2")
	integration.CompareJsonValue(t, artifactsConfigPath, "l2_config.denom", "uroll")
	integration.CompareJsonValue(t, artifactsConfigPath, "l2_config.moniker", "computer")
}
