//go:build integration
// +build integration

package cmd_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/common"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models/relayer"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/testutil"
)

func TestRelayerInit(t *testing.T) {
	ctx := weavecontext.NewAppContext(relayer.NewRelayerState())
	ctx = weavecontext.SetMinitiaHome(ctx, TestMinitiaHome)

	firstModel, err := relayer.NewRollupSelect(ctx)
	assert.Nil(t, err)

	miniEvmIdx := -1
	networks, err := registry.GetAllL2AvailableNetwork(registry.InitiaL1Testnet)
	if err != nil {
		t.Fatalf("get all l2 available networks: %v", err)
	}

	sort.Slice(networks, func(i, j int) bool {
		return strings.ToLower(networks[i].PrettyName) < strings.ToLower(networks[j].PrettyName)
	})
	for i, network := range networks {
		if strings.EqualFold(network.PrettyName, "Minievm") {
			miniEvmIdx = i
			break
		}
	}

	if miniEvmIdx == -1 {
		t.Fatalf("'Minievm' not found in networks")
	}

	pressDownSteps := make(testutil.Steps, miniEvmIdx)
	for i := 0; i < miniEvmIdx; i++ {
		pressDownSteps[i] = testutil.PressDown
	}

	steps := testutil.Steps{
		testutil.PressEnter,   // press enter to confirm selecting whitelisted rollups
		testutil.PressEnter,   // press enter to confirm selecting testnet
		testutil.WaitFetching, // wait fetching rollup networks
	}
	steps = append(steps, pressDownSteps...) // press down until its minievm
	steps = append(steps, testutil.Steps{
		testutil.PressEnter, // press enter to confirm selecting minievm
		testutil.PressSpace, // press space to select relaying all channels
		testutil.PressEnter, // press enter to confirm the selection
		testutil.WaitFor(func() bool {
			userHome, _ := os.UserHomeDir()
			if _, err := os.Stat(filepath.Join(userHome, common.HermesHome, "config.toml")); os.IsNotExist(err) {
				return false
			}
			return true
		}), // wait for relayer config to be created
		testutil.PressEnter,           // press enter to confirm generating new key on l1
		testutil.PressDown,            // press down once to select generate key on l2
		testutil.PressEnter,           // press enter to confirm the selection
		testutil.WaitFetching,         // wait for the generation of keys
		testutil.TypeText("continue"), // type to proceed after the mnemonic display page
		testutil.PressEnter,           // press enter to confirm the typing
		testutil.WaitFetching,         // wait for account balances fetching
		testutil.PressDown,            // press down once to move the selector
		testutil.PressDown,            // press down again to move the selector to skip the funding
		testutil.PressEnter,           // press enter to confirm the selection
	}...)

	finalModel := testutil.RunProgramWithSteps(t, firstModel, steps)

	// Check the final state here
	assert.IsType(t, &relayer.TerminalState{}, finalModel)

	if _, ok := finalModel.(*relayer.TerminalState); ok {
		assert.True(t, ok)
	}

}
