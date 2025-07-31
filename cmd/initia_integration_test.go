//go:build integration
// +build integration

package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/initia"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/testutil"
)

func setupInitia(t *testing.T) (string, tea.Model) {
	setup(t, []service.CommandName{service.UpgradableInitia})

	ctx := context.NewAppContext(initia.NewRunL1NodeState())

	homeDir, _ := os.Getwd()

	initiaHome := filepath.Join(homeDir, "initia.weave.test")
	ctx = context.SetInitiaHome(ctx, initiaHome)

	firstModel, _ := initia.NewRunL1NodeNetworkSelect(ctx)

	return initiaHome, firstModel
}

func teardownInitia(t *testing.T) {
	teardown(t, []service.CommandName{service.UpgradableInitia})
}

func TestInitiaInitTestnetNoSync(t *testing.T) {
	initiaHome, firstModel := setupInitia(t)
	defer teardownInitia(t)

	// Ensure that there is no previous Initia home
	_, err := os.Stat(initiaHome)
	assert.NotNil(t, err)

	steps := testutil.Steps{
		testutil.PressDown,           // press down once to select Testnet
		testutil.PressEnter,          // press enter to select Testnet
		testutil.TypeText("Moniker"), // type in the moniker
		testutil.PressEnter,          // press enter to confirm the moniker
		testutil.PressSpace,          // press space to enable REST
		testutil.PressEnter,          // press enter to confirm turning on only REST and not gRPC
		testutil.TypeText("3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656"), // type in the seeds
		testutil.PressEnter, // press enter to confirm the seeds
		testutil.TypeText("3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656"), // type in the persistent peers
		testutil.PressEnter, // press enter to confirm the persistent peers
		testutil.PressEnter, // press enter to confirm pruning default
		testutil.PressEnter, // press enter to confirm Allow Upgrade
		testutil.WaitFor(func() bool {
			if _, err := os.Stat(initiaHome); os.IsNotExist(err) {
				return false
			}
			return true
		}),
		testutil.WaitFetching, // wait for the Initia app to be created
		testutil.PressDown,    // press down twice to select No Sync
		testutil.PressDown,    //
		testutil.PressEnter,   // press enter to confirm selecting No Sync
	}

	finalModel := testutil.RunProgramWithSteps(t, firstModel, steps)
	defer testutil.ClearTestDir(initiaHome)

	// Check the final state here
	assert.IsType(t, &initia.TerminalState{}, finalModel)

	if _, ok := finalModel.(*initia.TerminalState); ok {
		assert.True(t, ok)
	}

	// Check if Initia home has been created
	_, err = os.Stat(initiaHome)
	assert.Nil(t, err)

	configTomlPath := filepath.Join(initiaHome, "config/config.toml")
	appTomlPath := filepath.Join(initiaHome, "config/app.toml")

	if _, err := os.Stat(configTomlPath); os.IsNotExist(err) {
		t.Fatalf("Config toml file does not exist: %s", configTomlPath)
	}

	if _, err := os.Stat(appTomlPath); os.IsNotExist(err) {
		t.Fatalf("App toml file does not exist: %s", appTomlPath)
	}

	// Assert values
	testutil.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
	testutil.CompareTomlValue(t, configTomlPath, "p2p.seeds", "3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656")
	testutil.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656")
	testutil.CompareTomlValue(t, configTomlPath, "statesync.enable", false)

	testutil.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0.015uinit")
	testutil.CompareTomlValue(t, appTomlPath, "api.enable", "true")
}

// TODO: Add test for state sync
// func TestInitiaInitTestnetStatesync(t *testing.T) {
// 	initiaHome, firstModel := setupInitia(t)
// 	defer teardownInitia(t)

// 	// Ensure that there is no previous Initia home
// 	_, err := os.Stat(initiaHome)
// 	assert.NotNil(t, err)

// 	steps := testutil.Steps{
// 		testutil.PressDown,           // press down once to select Testnet
// 		testutil.PressEnter,          // press enter to select Testnet
// 		testutil.TypeText("Moniker"), // type in the moniker
// 		testutil.PressEnter,          // press enter to confirm the moniker
// 		testutil.PressSpace,          // press space to enable REST
// 		testutil.PressEnter,          // press enter to confirm turning on only REST and not gRPC
// 		testutil.TypeText("3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656"), // type in the seeds
// 		testutil.PressEnter, // press enter to confirm the seeds
// 		testutil.TypeText("3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656"), // type in the persistent peers
// 		testutil.PressEnter, // press enter to confirm the persistent peers
// 		testutil.PressEnter, // press enter to confirm pruning default
// 		testutil.PressEnter, // press enter to confirm Allow Upgrade
// 		testutil.WaitFor(func() bool {
// 			if _, err := os.Stat(initiaHome); os.IsNotExist(err) {
// 				return false
// 			}
// 			return true
// 		}), // wait for the Initia app to be created
// 		testutil.PressDown,  // press down once to select State Sync
// 		testutil.PressEnter, // press enter to confirm selecting State Sync
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the checking of existing data
// 		testutil.PressEnter, // press enter to proceed with syncing
// 		testutil.TypeText("https://initia-testnet-rpc.polkachu.com:443"), // type in the state sync rpc
// 		testutil.PressEnter, // press enter to confirm the rpc
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the fetching of the default value
// 		testutil.TypeText("1d9b9512f925cf8808e7f76d71a788d82089fe76@65.108.198.118:25756"), // type in the additional peer for state sync
// 		testutil.PressEnter, // press enter to confirm the peer
// 	}

// 	finalModel := testutil.RunProgramWithSteps(t, firstModel, steps)
// 	defer testutil.ClearTestDir(initiaHome)

// 	// Check the final state here
// 	assert.IsType(t, &initia.TerminalState{}, finalModel)

// 	if _, ok := finalModel.(*initia.TerminalState); ok {
// 		assert.True(t, ok)
// 	}

// 	// Check if Initia home has been created
// 	_, err = os.Stat(initiaHome)
// 	assert.Nil(t, err)

// 	configTomlPath := filepath.Join(initiaHome, "config/config.toml")
// 	appTomlPath := filepath.Join(initiaHome, "config/app.toml")

// 	if _, err := os.Stat(configTomlPath); os.IsNotExist(err) {
// 		t.Fatalf("Config toml file does not exist: %s", configTomlPath)
// 	}

// 	if _, err := os.Stat(appTomlPath); os.IsNotExist(err) {
// 		t.Fatalf("App toml file does not exist: %s", appTomlPath)
// 	}

// 	// Assert values
// 	testutil.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.seeds", "3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656,1d9b9512f925cf8808e7f76d71a788d82089fe76@65.108.198.118:25756")
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.enable", "true")
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "https://initia-testnet-rpc.polkachu.com:443,https://initia-testnet-rpc.polkachu.com:443")

// 	testutil.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0.015uinit")
// 	testutil.CompareTomlValue(t, appTomlPath, "api.enable", "true")
// }

// func TestInitiaInitLocal(t *testing.T) {
// 	ctx := context.NewAppContext(initia.NewRunL1NodeState())
// 	initiaHome := TestInitiaHome + ".local"
// 	ctx = context.SetInitiaHome(ctx, initiaHome)

// 	firstModel := initia.NewRunL1NodeNetworkSelect(ctx)

// 	// Ensure that there is no previous Initia home
// 	_, err := os.Stat(initiaHome)
// 	assert.NotNil(t, err)

// 	steps := []testutil.Step{
// 		testutil.PressDown,  // press down once to select Local
// 		testutil.PressEnter, // press enter to confirm selecting Local
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the fetching of Initia versions
// 		testutil.PressEnter,            // press enter to select the latest version available
// 		testutil.TypeText("ChainId-1"), // type in the chain id
// 		testutil.PressEnter,            // press enter to confirm the chain id
// 		testutil.TypeText("Moniker"),   // type in the moniker
// 		testutil.PressEnter,            // press enter to confirm the moniker
// 		testutil.TypeText("0uinit"),    // type in the gas price
// 		testutil.PressEnter,            // press enter to confirm the minimum gas price
// 		testutil.PressEnter,            // press enter to disable both REST and gRPC
// 		testutil.PressEnter,            // press enter to skip adding seeds
// 		testutil.PressEnter,            // press enter to skip adding persistent peers
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}),
// testutil.PressEnter, // press enter to confirm pruning default
// 		testutil.PressDown,  // press down once to select disallow upgrade
// 		testutil.PressEnter, // press enter to confirm
// 		testutil.WaitFor(func() bool {
// 			if _, err := os.Stat(initiaHome); os.IsNotExist(err) {
// 				return false
// 			}
// 			return true
// 		}), // wait for the app to be created
// 	}

// 	finalModel := testutil.RunProgramWithSteps(t, firstModel, steps)
// 	defer testutil.ClearTestDir(initiaHome)

// 	// Check the final state here
// 	assert.IsType(t, &initia.InitializingAppLoading{}, finalModel)

// 	if _, ok := finalModel.(*initia.InitializingAppLoading); ok {
// 		assert.True(t, ok)
// 	}

// 	// Check if Initia home has been created
// 	_, err = os.Stat(initiaHome)
// 	assert.Nil(t, err)

// 	configTomlPath := filepath.Join(initiaHome, "config/config.toml")
// 	appTomlPath := filepath.Join(initiaHome, "config/app.toml")

// 	if _, err := os.Stat(configTomlPath); os.IsNotExist(err) {
// 		t.Fatalf("Config toml file does not exist: %s", configTomlPath)
// 	}

// 	if _, err := os.Stat(appTomlPath); os.IsNotExist(err) {
// 		t.Fatalf("App toml file does not exist: %s", appTomlPath)
// 	}

// 	// Assert values
// 	testutil.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.enable", false)
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "")

// 	testutil.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0uinit")
// 	testutil.CompareTomlValue(t, appTomlPath, "api.enable", "false")
// }

// func TestInitiaInitLocalExisting(t *testing.T) {
// 	ctx := context.NewAppContext(initia.NewRunL1NodeState())
// 	initiaHome := TestInitiaHome + ".local.existing"
// 	ctx = context.SetInitiaHome(ctx, initiaHome)

// 	firstModel := initia.NewRunL1NodeNetworkSelect(ctx)

// 	// Ensure that there is no previous Initia home
// 	_, err := os.Stat(initiaHome)
// 	assert.NotNil(t, err)

// 	steps := []testutil.Step{
// 		testutil.PressDown,  // press down to select Local
// 		testutil.PressEnter, // press enter to confirm selecting Local
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the fetching of Initia versions
// 		testutil.PressEnter,            // press enter to select the latest version available
// 		testutil.TypeText("ChainId-1"), // type in the chain id
// 		testutil.PressEnter,            // press enter to confirm the chain id
// 		testutil.TypeText("Moniker"),   // type in the moniker
// 		testutil.PressEnter,            // press enter to confirm the moniker
// 		testutil.TypeText("0uinit"),    // type in the minimum gas price
// 		testutil.PressEnter,            // press enter to confirm the minimum gas price
// 		testutil.PressEnter,            // press enter to disable both REST and gRPC
// 		testutil.PressEnter,            // press enter to skip adding the seeds
// 		testutil.PressEnter,            // press enter to skip adding the persistent peers
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the app to be created
// testutil.PressEnter, // press enter to confirm pruning default
// 		testutil.PressEnter, // press enter to confirm allow upgrade
// 		testutil.WaitFor(func() bool {
// 			if _, err := os.Stat(initiaHome); os.IsNotExist(err) {
// 				return false
// 			}
// 			return true
// 		}), // wait for the app to be created
// 	}

// 	finalModel := testutil.RunProgramWithSteps(t, firstModel, steps)

// 	// Check the final state here
// 	assert.IsType(t, &initia.InitializingAppLoading{}, finalModel)

// 	if _, ok := finalModel.(*initia.InitializingAppLoading); ok {
// 		assert.True(t, ok)
// 	}

// 	// Check if Initia home has been created
// 	_, err = os.Stat(initiaHome)
// 	assert.Nil(t, err)

// 	configTomlPath := filepath.Join(initiaHome, "config/config.toml")
// 	appTomlPath := filepath.Join(initiaHome, "config/app.toml")

// 	if _, err := os.Stat(configTomlPath); os.IsNotExist(err) {
// 		t.Fatalf("Config toml file does not exist: %s", configTomlPath)
// 	}

// 	if _, err := os.Stat(appTomlPath); os.IsNotExist(err) {
// 		t.Fatalf("App toml file does not exist: %s", appTomlPath)
// 	}

// 	// Assert values
// 	testutil.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.enable", false)
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "")

// 	testutil.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0uinit")
// 	testutil.CompareTomlValue(t, appTomlPath, "api.enable", "false")

// 	ctx = context.NewAppContext(initia.NewRunL1NodeState())
// 	ctx = context.SetInitiaHome(ctx, initiaHome)

// 	firstModel = initia.NewRunL1NodeNetworkSelect(ctx)

// 	// Ensure that there is an existing Initia home
// 	_, err = os.Stat(initiaHome)
// 	assert.Nil(t, err)

// 	steps = []testutil.Step{
// 		testutil.PressDown,  // press down once to select Local
// 		testutil.PressEnter, // press enter to confirm selecting Local
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the fetching of Initia versions
// 		testutil.PressEnter,            // press enter to select the latest version available
// 		testutil.TypeText("ChainId-2"), // type in the chain id
// 		testutil.PressEnter,            // press enter to confirm the chain id
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the checking of existing config
// 		testutil.PressEnter, // press enter to use current files
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the checking of existing genesis
// 		testutil.PressEnter, // press enter to use the current genesis
// testutil.PressEnter, // press enter to confirm pruning default
// 		testutil.PressEnter, // press enter to comfirm allow upgrade
// 	}

// 	finalModel = testutil.RunProgramWithSteps(t, firstModel, steps)

// 	testutil.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.enable", false)
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "")

// 	testutil.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0uinit")
// 	testutil.CompareTomlValue(t, appTomlPath, "api.enable", "false")

// 	ctx = context.NewAppContext(initia.NewRunL1NodeState())
// 	ctx = context.SetInitiaHome(ctx, initiaHome)

// 	firstModel = initia.NewRunL1NodeNetworkSelect(ctx)

// 	// Ensure that there is an existing Initia home
// 	_, err = os.Stat(initiaHome)
// 	assert.Nil(t, err)

// 	steps = testutil.Steps{
// 		testutil.PressDown,  // press down once to select Local
// 		testutil.PressEnter, // press enter to confirm selecting Local
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the fetching of Initia versions
// 		testutil.PressEnter,            // press enter to select the latest version available
// 		testutil.TypeText("ChainId-3"), // type in the chain id
// 		testutil.PressEnter,            // press enter to confirm the chain id
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the checking of existing config
// 		testutil.PressUp,                // press up once to select replacing the config
// 		testutil.PressEnter,             // press enter to confirm replacing the config
// 		testutil.TypeText("NewMoniker"), // type in the new moniker
// 		testutil.PressEnter,             // press enter to confirm the new moniker
// 		testutil.TypeText("0.015uinit"), // type in the new minimum gas price
// 		testutil.PressEnter,             // press enter to confirm the new minimum gas price
// 		testutil.PressSpace,             // press space to enable REST
// 		testutil.PressDown,              // press down once to move the cursor to gRPC
// 		testutil.PressSpace,             // press space to also enable gRPC
// 		testutil.PressEnter,             // press enter to confirm enabling both REST and gRPC
// 		testutil.PressEnter,             // press enter to skip adding the seeds
// 		testutil.PressEnter,             // press enter to skip adding the persistent peers
// 		testutil.WaitFor(func() bool {
// 			return true
// 		}), // wait for the checking of existing genesis
// 		testutil.PressUp,    // press up once to select replacing the genesis
// 		testutil.PressEnter, // press enter to confirm replacing the genesis
// testutil.PressEnter, // press enter to confirm pruning default
// 		testutil.PressEnter, // press enter to comfirm allow upgrade
// 	}

// 	finalModel = testutil.RunProgramWithSteps(t, firstModel, steps)
// 	defer testutil.ClearTestDir(initiaHome)

// 	testutil.CompareTomlValue(t, configTomlPath, "moniker", "NewMoniker")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
// 	testutil.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.enable", false)
// 	testutil.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "")

// 	testutil.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0.015uinit")
// 	testutil.CompareTomlValue(t, appTomlPath, "api.enable", "true")
// }
