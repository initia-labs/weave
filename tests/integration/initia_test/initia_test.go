package initia_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/initia"
	"github.com/initia-labs/weave/tests/integration"
)

const (
	TestInitiaHome = ".initia.weave.test"
)

func TestInitiaInitTestnetNoSync(t *testing.T) {
	t.Parallel()
	ctx := context.NewAppContext(initia.NewRunL1NodeState())
	initiaHome := TestInitiaHome + ".nosync"
	ctx = context.SetInitiaHome(ctx, initiaHome)

	firstModel := initia.NewRunL1NodeNetworkSelect(ctx)

	// Ensure that there is no previous Initia home
	_, err := os.Stat(initiaHome)
	assert.NotNil(t, err)

	steps := integration.Steps{
		integration.PressEnter,          // press enter to select Testnet
		integration.TypeText("Moniker"), // type in the moniker
		integration.PressEnter,          // press enter to confirm the moniker
		integration.PressSpace,          // press space to enable REST
		integration.PressEnter,          // press enter to confirm turning on only REST and not gRPC
		integration.TypeText("3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656"), // type in the seeds
		integration.PressEnter, // press enter to confirm the seeds
		integration.TypeText("3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656"), // type in the persistent peers
		integration.PressEnter, // press enter to confirm the persistent peers
		integration.WaitFor(func() bool {
			if _, err := os.Stat(initiaHome); os.IsNotExist(err) {
				return false
			}
			return true
		}), // wait for the Initia app to be created
		integration.PressDown,  // press down twice to select No Sync
		integration.PressDown,  //
		integration.PressEnter, // press enter to confirm selecting No Sync
	}

	finalModel := integration.RunProgramWithSteps(t, firstModel, steps)
	defer integration.ClearTestDir(initiaHome)

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
	integration.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
	integration.CompareTomlValue(t, configTomlPath, "p2p.seeds", "3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656")
	integration.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656")
	integration.CompareTomlValue(t, configTomlPath, "statesync.enable", false)

	integration.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0.15uinit")
	integration.CompareTomlValue(t, appTomlPath, "api.enable", "true")
}

func TestInitiaInitTestnetStatesync(t *testing.T) {
	t.Skip("Skipping initia init with state sync test")
	ctx := context.NewAppContext(initia.NewRunL1NodeState())
	initiaHome := TestInitiaHome + ".statesync"
	ctx = context.SetInitiaHome(ctx, initiaHome)

	firstModel := initia.NewRunL1NodeNetworkSelect(ctx)

	// Ensure that there is no previous Initia home
	_, err := os.Stat(initiaHome)
	assert.NotNil(t, err)

	steps := integration.Steps{
		integration.PressEnter,          // press enter to select Testnet
		integration.TypeText("Moniker"), // type in the moniker
		integration.PressEnter,          // press enter to confirm the moniker
		integration.PressSpace,          // press space to enable REST
		integration.PressEnter,          // press enter to confirm turning on only REST and not gRPC
		integration.TypeText("3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656"), // type in the seeds
		integration.PressEnter, // press enter to confirm the seeds
		integration.TypeText("3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656"), // type in the persistent peers
		integration.PressEnter, // press enter to confirm the persistent peers
		integration.WaitFor(func() bool {
			if _, err := os.Stat(initiaHome); os.IsNotExist(err) {
				return false
			}
			return true
		}), // wait for the Initia app to be created
		integration.PressDown,  // press down once to select State Sync
		integration.PressEnter, // press enter to confirm selecting State Sync
		integration.WaitFor(func() bool {
			return true
		}),                                                                  // wait for the checking of existing data
		integration.PressEnter,                                              // press enter to proceed with syncing
		integration.TypeText("https://initia-testnet-rpc.polkachu.com:443"), // type in the state sync rpc
		integration.PressEnter,                                              // press enter to confirm the rpc
		integration.WaitFor(func() bool {
			return true
		}), // wait for the fetching of the default value
		integration.TypeText("1d9b9512f925cf8808e7f76d71a788d82089fe76@65.108.198.118:25756"), // type in the additional peer for state sync
		integration.PressEnter, // press enter to confirm the peer
	}

	finalModel := integration.RunProgramWithSteps(t, firstModel, steps)
	defer integration.ClearTestDir(initiaHome)

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
	integration.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
	integration.CompareTomlValue(t, configTomlPath, "p2p.seeds", "3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656")
	integration.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "3715cdb41efb45714eb534c3943c5947f4894787@34.143.179.242:26656,1d9b9512f925cf8808e7f76d71a788d82089fe76@65.108.198.118:25756")
	integration.CompareTomlValue(t, configTomlPath, "statesync.enable", "true")
	integration.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "https://initia-testnet-rpc.polkachu.com:443,https://initia-testnet-rpc.polkachu.com:443")

	integration.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0.15uinit")
	integration.CompareTomlValue(t, appTomlPath, "api.enable", "true")
}

func TestInitiaInitLocal(t *testing.T) {
	t.Parallel()
	ctx := context.NewAppContext(initia.NewRunL1NodeState())
	initiaHome := TestInitiaHome + ".local"
	ctx = context.SetInitiaHome(ctx, initiaHome)

	firstModel := initia.NewRunL1NodeNetworkSelect(ctx)

	// Ensure that there is no previous Initia home
	_, err := os.Stat(initiaHome)
	assert.NotNil(t, err)

	steps := []integration.Step{
		integration.PressDown,  // press down once to select Local
		integration.PressEnter, // press enter to confirm selecting Local
		integration.WaitFor(func() bool {
			return true
		}),                                // wait for the fetching of Initia versions
		integration.PressEnter,            // press enter to select the latest version available
		integration.TypeText("ChainId-1"), // type in the chain id
		integration.PressEnter,            // press enter to confirm the chain id
		integration.TypeText("Moniker"),   // type in the moniker
		integration.PressEnter,            // press enter to confirm the moniker
		integration.TypeText("0uinit"),    // type in the gas price
		integration.PressEnter,            // press enter to confirm the minimum gas price
		integration.PressEnter,            // press enter to disable both REST and gRPC
		integration.PressEnter,            // press enter to skip adding seeds
		integration.PressEnter,            // press enter to skip adding persistent peers
		integration.WaitFor(func() bool {
			if _, err := os.Stat(initiaHome); os.IsNotExist(err) {
				return false
			}
			return true
		}), // wait for the app to be created
	}

	finalModel := integration.RunProgramWithSteps(t, firstModel, steps)
	defer integration.ClearTestDir(initiaHome)

	// Check the final state here
	assert.IsType(t, &initia.InitializingAppLoading{}, finalModel)

	if _, ok := finalModel.(*initia.InitializingAppLoading); ok {
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
	integration.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
	integration.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
	integration.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
	integration.CompareTomlValue(t, configTomlPath, "statesync.enable", false)
	integration.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "")

	integration.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0uinit")
	integration.CompareTomlValue(t, appTomlPath, "api.enable", "false")
}

func TestInitiaInitLocalExisting(t *testing.T) {
	t.Parallel()
	ctx := context.NewAppContext(initia.NewRunL1NodeState())
	initiaHome := TestInitiaHome + ".local.existing"
	ctx = context.SetInitiaHome(ctx, initiaHome)

	firstModel := initia.NewRunL1NodeNetworkSelect(ctx)

	// Ensure that there is no previous Initia home
	_, err := os.Stat(initiaHome)
	assert.NotNil(t, err)

	steps := []integration.Step{
		integration.PressDown,  // press down to select Local
		integration.PressEnter, // press enter to confirm selecting Local
		integration.WaitFor(func() bool {
			return true
		}),                                // wait for the fetching of Initia versions
		integration.PressEnter,            // press enter to select the latest version available
		integration.TypeText("ChainId-1"), // type in the chain id
		integration.PressEnter,            // press enter to confirm the chain id
		integration.TypeText("Moniker"),   // type in the moniker
		integration.PressEnter,            // press enter to confirm the moniker
		integration.TypeText("0uinit"),    // type in the minimum gas price
		integration.PressEnter,            // press enter to confirm the minimum gas price
		integration.PressEnter,            // press enter to disable both REST and gRPC
		integration.PressEnter,            // press enter to skip adding the seeds
		integration.PressEnter,            // press enter to skip adding the persistent peers
		integration.WaitFor(func() bool {
			if _, err := os.Stat(initiaHome); os.IsNotExist(err) {
				return false
			}
			return true
		}), // wait for the app to be created
	}

	finalModel := integration.RunProgramWithSteps(t, firstModel, steps)

	// Check the final state here
	assert.IsType(t, &initia.InitializingAppLoading{}, finalModel)

	if _, ok := finalModel.(*initia.InitializingAppLoading); ok {
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
	integration.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
	integration.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
	integration.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
	integration.CompareTomlValue(t, configTomlPath, "statesync.enable", false)
	integration.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "")

	integration.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0uinit")
	integration.CompareTomlValue(t, appTomlPath, "api.enable", "false")

	ctx = context.NewAppContext(initia.NewRunL1NodeState())
	ctx = context.SetInitiaHome(ctx, initiaHome)

	firstModel = initia.NewRunL1NodeNetworkSelect(ctx)

	// Ensure that there is an existing Initia home
	_, err = os.Stat(initiaHome)
	assert.Nil(t, err)

	steps = []integration.Step{
		integration.PressDown,  // press down once to select Local
		integration.PressEnter, // press enter to confirm selecting Local
		integration.WaitFor(func() bool {
			return true
		}),                                // wait for the fetching of Initia versions
		integration.PressEnter,            // press enter to select the latest version available
		integration.TypeText("ChainId-2"), // type in the chain id
		integration.PressEnter,            // press enter to confirm the chain id
		integration.WaitFor(func() bool {
			return true
		}),                     // wait for the checking of existing config
		integration.PressEnter, // press enter to use current files
		integration.WaitFor(func() bool {
			return true
		}),                     // wait for the checking of existing genesis
		integration.PressEnter, // press enter to use the current genesis
	}

	finalModel = integration.RunProgramWithSteps(t, firstModel, steps)

	integration.CompareTomlValue(t, configTomlPath, "moniker", "Moniker")
	integration.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
	integration.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
	integration.CompareTomlValue(t, configTomlPath, "statesync.enable", false)
	integration.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "")

	integration.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0uinit")
	integration.CompareTomlValue(t, appTomlPath, "api.enable", "false")

	ctx = context.NewAppContext(initia.NewRunL1NodeState())
	ctx = context.SetInitiaHome(ctx, initiaHome)

	firstModel = initia.NewRunL1NodeNetworkSelect(ctx)

	// Ensure that there is an existing Initia home
	_, err = os.Stat(initiaHome)
	assert.Nil(t, err)

	steps = integration.Steps{
		integration.PressDown,  // press down once to select Local
		integration.PressEnter, // press enter to confirm selecting Local
		integration.WaitFor(func() bool {
			return true
		}),                                // wait for the fetching of Initia versions
		integration.PressEnter,            // press enter to select the latest version available
		integration.TypeText("ChainId-3"), // type in the chain id
		integration.PressEnter,            // press enter to confirm the chain id
		integration.WaitFor(func() bool {
			return true
		}),                                 // wait for the checking of existing config
		integration.PressUp,                // press up once to select replacing the config
		integration.PressEnter,             // press enter to confirm replacing the config
		integration.TypeText("NewMoniker"), // type in the new moniker
		integration.PressEnter,             // press enter to confirm the new moniker
		integration.TypeText("0.015uinit"), // type in the new minimum gas price
		integration.PressEnter,             // press enter to confirm the new minimum gas price
		integration.PressSpace,             // press space to enable REST
		integration.PressDown,              // press down once to move the cursor to gRPC
		integration.PressSpace,             // press space to also enable gRPC
		integration.PressEnter,             // press enter to confirm enabling both REST and gRPC
		integration.PressEnter,             // press enter to skip adding the seeds
		integration.PressEnter,             // press enter to skip adding the persistent peers
		integration.WaitFor(func() bool {
			return true
		}),                     // wait for the checking of existing genesis
		integration.PressUp,    // press up once to select replacing the genesis
		integration.PressEnter, // press enter to confirm replacing the genesis
	}

	finalModel = integration.RunProgramWithSteps(t, firstModel, steps)
	defer integration.ClearTestDir(initiaHome)

	integration.CompareTomlValue(t, configTomlPath, "moniker", "NewMoniker")
	integration.CompareTomlValue(t, configTomlPath, "p2p.seeds", "")
	integration.CompareTomlValue(t, configTomlPath, "p2p.persistent_peers", "")
	integration.CompareTomlValue(t, configTomlPath, "statesync.enable", false)
	integration.CompareTomlValue(t, configTomlPath, "statesync.rpc_servers", "")

	integration.CompareTomlValue(t, appTomlPath, "minimum-gas-prices", "0.015uinit")
	integration.CompareTomlValue(t, appTomlPath, "api.enable", "true")
}
