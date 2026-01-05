//go:build integration
// +build integration

package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/opinit_bots"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/testutil"
)

func setupOPInit(t *testing.T) context.Context {
	setup(t, []service.CommandName{service.OPinitExecutor, service.OPinitChallenger})

	return prepareContext()
}

func teardownOPInit(t *testing.T) {
	teardown(t, []service.CommandName{service.OPinitExecutor, service.OPinitChallenger})
}

func prepareContext() context.Context {
	ctx := weavecontext.NewAppContext(opinit_bots.NewOPInitBotsState())
	ctx = weavecontext.SetMinitiaHome(ctx, TestMinitiaHome)
	ctx = weavecontext.SetOPInitHome(ctx, TestOPInitHome)
	return ctx
}

func setupKeys(t *testing.T) func() {
	ctx := prepareContext()
	firstModel := opinit_bots.NewEnsureOPInitBotsBinaryLoadingModel(
		ctx,
		func(nextCtx context.Context) (tea.Model, error) {
			return opinit_bots.ProcessMinitiaConfig(nextCtx, opinit_bots.NewSetupBotCheckbox)
		},
	)

	// Ensure that there is no previous OPInit home
	_, err := os.Stat(TestOPInitHome)
	assert.NotNil(t, err)

	steps := []testutil.Step{
		testutil.WaitFor(func() bool {
			userHome, _ := os.UserHomeDir()
			if _, err := os.Stat(filepath.Join(userHome, common.WeaveDirectory, "data/opinitd")); os.IsNotExist(err) {
				return false
			}
			return true
		}),
		testutil.PressSpace,
		testutil.PressDown,
		testutil.PressSpace,
		testutil.PressDown,
		testutil.PressSpace,
		testutil.PressDown,
		testutil.PressSpace,
		testutil.PressEnter,
		testutil.PressEnter,
		testutil.PressEnter,
		testutil.PressEnter,
		testutil.PressEnter,
		testutil.PressEnter,
	}

	finalModel := testutil.RunProgramWithSteps(t, firstModel, steps)

	// Check the final state here
	assert.IsType(t, &opinit_bots.TerminalState{}, finalModel)

	if _, ok := finalModel.(*opinit_bots.TerminalState); ok {
		assert.True(t, ok)
	}

	// Check if OPInit home has been created
	_, err = os.Stat(TestOPInitHome)
	assert.Nil(t, err)

	// Check the keys have been created
	_, err = os.Stat(filepath.Join(TestOPInitHome, "weave-dummy/keyring-test/weave_batch_submitter.info"))
	assert.Nil(t, err)

	_, err = os.Stat(filepath.Join(TestOPInitHome, "weave-dummy/keyring-test/weave_bridge_executor.info"))
	assert.Nil(t, err)

	_, err = os.Stat(filepath.Join(TestOPInitHome, "weave-dummy/keyring-test/weave_challenger.info"))
	assert.Nil(t, err)

	_, err = os.Stat(filepath.Join(TestOPInitHome, "weave-dummy/keyring-test/weave_output_submitter.info"))
	assert.Nil(t, err)

	return func() { testutil.ClearTestDir(TestOPInitHome) }
}

func TestOPInitBotsSetup(t *testing.T) {
	cleanup := setupKeys(t)
	defer cleanup()
}

func TestOPInitBotsInit(t *testing.T) {
	cleanup := setupKeys(t)
	defer cleanup()

	ctx := setupOPInit(t)
	defer teardownOPInit(t)

	firstModel := opinit_bots.NewEnsureOPInitBotsBinaryLoadingModel(
		ctx,
		func(nextCtx context.Context) (tea.Model, error) {
			return opinit_bots.ProcessMinitiaConfig(nextCtx, opinit_bots.OPInitBotInitSelectExecutor)
		},
	)

	steps := []testutil.Step{
		testutil.PressEnter,         // press enter to init executor bot
		testutil.WaitFetching,       // wait checking for the existing rollup app
		testutil.PressEnter,         // press enter to select testnet as l1
		testutil.WaitFetching,       // wait checking for the existing rollup app
		testutil.PressTab,           // press tab to use the default listen address
		testutil.PressEnter,         // press enter to confirm using the default listen address
		testutil.PressEnter,         // press enter to confirm using the default l1 rpc endpoint
		testutil.TypeText("move-1"), // type in the rollup chain id
		testutil.PressEnter,         // press enter to confirm the rollup chain id
		testutil.TypeText("https://rpc-move-1.anvil.asia-southeast.initia.xyz"), // press tab to use the default rollup rpc endpoint
		testutil.PressEnter,    // press enter to confirm using the default rollup rpc endpoint
		testutil.PressTab,      // press tab to use the default gas denom
		testutil.PressEnter,    // press enter to confirm using the default gas denom
		testutil.WaitFetching,  // wait checking for the existing rollup app
		testutil.WaitFetching,  // wait checking for the existing rollup app
		testutil.TypeText("1"), // type L1 start height
		testutil.PressEnter,    // press enter to confirm l1 start height
	}

	finalModel := testutil.RunProgramWithSteps(t, firstModel, steps)

	// Check the final state here
	assert.IsType(t, &opinit_bots.OPinitBotSuccessful{}, finalModel)
}
