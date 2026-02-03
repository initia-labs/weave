//go:build integration
// +build integration

package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/minitia"
	"github.com/initia-labs/weave/testutil"
)

func TestMinitiaLaunchWithExisting(t *testing.T) {
	t.Skip("Skipping minitia launch with existing test")
	_ = testutil.SetupGasStation(t)

	ctx := context.NewAppContext(*minitia.NewLaunchState())
	ctx = context.SetMinitiaHome(ctx, TestMinitiaHome)

	firstModel := minitia.NewExistingMinitiaChecker(ctx)

	// Ensure that there is no previous Minitia home
	_, err := os.Stat(TestMinitiaHome)
	assert.NotNil(t, err)

	steps := testutil.Steps{
		testutil.WaitFetching,                   // wait checking for the existing rollup app
		testutil.PressEnter,                     // press enter to select Testnet
		testutil.PressEnter,                     // press enter to select Move vm for the rollup
		testutil.WaitFetching,                   // wait for the fetching of the latest Move rollup version
		testutil.TypeText("rollup-1"),           // type in the rollup chain id
		testutil.PressEnter,                     // press enter to confirm the rollup chain id
		testutil.PressTab,                       // press tab to use the default gas denom
		testutil.PressEnter,                     // press enter to confirm using the default gas denom
		testutil.PressTab,                       // press tab to use the default moniker
		testutil.PressEnter,                     // press enter to confirm using the default moniker
		testutil.PressTab,                       // press tab to use the default submission interval
		testutil.PressEnter,                     // press enter to confirm using the default submission interval
		testutil.PressTab,                       // press tab to use the default output submission period
		testutil.PressEnter,                     // press enter to confirm using the default output submission period
		testutil.PressDown,                      // press down once to select Initia L1 as the DA layer
		testutil.PressEnter,                     // press enter to confirm selecting Initia L1 as the DA layer
		testutil.PressEnter,                     // press enter to confirm enabling the oracle
		testutil.PressEnter,                     // press enter to generate keys for system keys
		testutil.WaitFetching,                   // wait for the fetching of gas station
		testutil.PressDown,                      // press down once to select filling each account balance manually
		testutil.PressEnter,                     // press enter to confirm selecting manual
		testutil.TypeText("1234567"),            // type in the balance for the bridge executor
		testutil.PressEnter,                     // press enter to confirm the balance for the bridge executor
		testutil.TypeText("1000000"),            // type in the balance for the output submitter
		testutil.PressEnter,                     // press enter to confirm the balance for the output submitter
		testutil.TypeText("1111111"),            // type in the balance for the batch submitter
		testutil.PressEnter,                     // press enter to confirm the balance for the batch submitter
		testutil.TypeText("234567"),             // type in the balance for the challenger
		testutil.PressEnter,                     // press enter to confirm the balance for the challenger
		testutil.TypeText("123456789123456789"), // type in the balance for the L2 operator
		testutil.PressEnter,                     // press enter to confirm the balance for the L2 operator
		testutil.TypeText("123456789123456789"), // type in the balance for the L2 bridge executor
		testutil.PressEnter,                     // press enter to confirm the balance for the L2 bridge executor
		testutil.PressDown,                      // press down to select do not add more genesis account
		testutil.PressEnter,                     // press enter to confirm the selection
		testutil.WaitFetching,                   // wait for the downloading of the Move rollup binary
		testutil.WaitFetching,                   // wait for the key generation
		testutil.TypeText("continue"),           // type in to continue the process
		testutil.PressEnter,                     // press enter to confirm continuing
		testutil.TypeText("y"),                  // type in y to confirm the broadcast of transactions
		testutil.PressEnter,                     // press enter to confirm transactions broadcast
	}

	finalModel := testutil.RunProgramWithSteps(t, firstModel, steps)

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
	testutil.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
	testutil.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
	testutil.CompareTomlValue(t, configTomlPath, "statesync.enable", false)

	testutil.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0umin")
	testutil.CompareTomlValue(t, appTomlPath, "api.enable", true)

	artifactsConfigPath := filepath.Join(TestMinitiaHome, "artifacts/config.json")

	if _, err := os.Stat(artifactsConfigPath); os.IsNotExist(err) {
		t.Fatalf("Artifacts config json file does not exist: %s", artifactsConfigPath)
	}

	testutil.CompareJsonValue(t, artifactsConfigPath, "l2_config.chain_id", "rollup-1")
	testutil.CompareJsonValue(t, artifactsConfigPath, "l2_config.denom", "umin")
	testutil.CompareJsonValue(t, artifactsConfigPath, "l2_config.moniker", "operator")
	testutil.CompareJsonValue(t, artifactsConfigPath, "op_bridge.enable_oracle", true)

	// Try again with existing Minitia home
	ctx = context.NewAppContext(*minitia.NewLaunchState())
	ctx = context.SetMinitiaHome(ctx, TestMinitiaHome)

	firstModel = minitia.NewExistingMinitiaChecker(ctx)

	// Ensure that there is an existing Minitia home
	_, err = os.Stat(TestMinitiaHome)
	assert.Nil(t, err)

	steps = testutil.Steps{
		testutil.WaitFetching,         // wait checking for the existing rollup app
		testutil.TypeText("confirm"),  // type in confirm to confirm removing the existing rollup app
		testutil.PressEnter,           // press enter to confirm removing the rollup app
		testutil.PressEnter,           // press enter to select Testnet
		testutil.PressDown,            // press down once to select the Wasm rollup
		testutil.PressEnter,           // press enter to confirm selecting the Wasm rollup
		testutil.WaitFetching,         // wait for the fetching of the latest Move rollup version
		testutil.TypeText("rollup-2"), // type in the rollup chain id
		testutil.PressEnter,           // press enter to confirm the rollup chain id
		testutil.TypeText("uroll"),    // type in the gas denom
		testutil.PressEnter,           // press enter to confirm the gas denom
		testutil.TypeText("computer"), // type in the moniker
		testutil.PressEnter,           // press enter to confirm the moniker
		testutil.PressTab,             // press tab to use the default submission interval
		testutil.PressEnter,           // press enter to confirm using the default submission interval
		testutil.PressTab,             // press tab to use the default output submission period
		testutil.PressEnter,           // press enter to confirm using the default output submission period
		testutil.PressEnter,           // press enter to confirm enabling the oracle
		testutil.PressDown,            // press down once to select importing existing keys
		testutil.PressEnter,           // press enter to confirm importing existing keys
		testutil.TypeText("lonely fly lend protect mix order legal organ fruit donkey dog state"), // type in the mnemonic for the operator
		testutil.PressEnter, // press enter to confirm the mnemonic
		testutil.TypeText("boy salmon resist afford dog cereal first myth require enough sunny cargo"), // type in the mnemonic for the bridge executor
		testutil.PressEnter, // press enter to confirm the mnemonic
		testutil.TypeText("young diagram garment finish barrel output pledge borrow tonight frozen clerk sadness"), // type in the mnemonic for the output submitter
		testutil.PressEnter, // press enter to confirm the mnemonic
		testutil.TypeText("patrol search opera diary hidden giggle crisp together toy print lemon very"), // type in the mnemonic for the batch submitter
		testutil.PressEnter, // press enter to confirm the mnemonic
		testutil.TypeText("door radar exhibit equip mom beach drift harbor tomorrow tree long stereo"), // type in the mnemonic for the challenger
		testutil.PressEnter,   // press enter to confirm the mnemonic
		testutil.WaitFetching, // wait for the fetching of gas station
		testutil.PressEnter,   // press enter to confirm using the default funding preset
		testutil.PressEnter,   // press enter to confirm adding a genesis account
		testutil.TypeText("init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr"), // type in the genesis account address
		testutil.PressEnter,             // press enter to confirm the address
		testutil.TypeText("1234567890"), // type in the genesis account initial balance
		testutil.PressEnter,             // press enter to confirm the balance
		testutil.PressDown,              // press down once to select not adding more accounts
		testutil.PressEnter,             // press enter to confirm the selection
		testutil.WaitFetching,           // wait for the downloading of the Wasm rollup binary
		testutil.WaitFetching,           // wait for the key recovery
		testutil.TypeText("y"),          // type in y to confirm the broadcast of transactions
		testutil.PressEnter,             // press enter to confirm transactions broadcast
	}

	finalModel = testutil.RunProgramWithSteps(t, firstModel, steps)
	defer testutil.ClearTestDir(TestMinitiaHome)

	// Assert values
	testutil.CompareJsonValue(t, artifactsConfigPath, "l2_config.chain_id", "rollup-2")
	testutil.CompareJsonValue(t, artifactsConfigPath, "l2_config.denom", "uroll")
	testutil.CompareJsonValue(t, artifactsConfigPath, "l2_config.moniker", "computer")
}
