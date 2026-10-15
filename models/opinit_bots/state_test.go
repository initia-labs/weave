package opinit_bots

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/stretchr/testify/require"
)

// Catches DA flags lost by page snapshots and shared DA map mutation.
func TestDAFlowCloneAndReselect(t *testing.T) {
	daFlowFixtures(t)
	state := daTestState()
	state.daIsCelestia = true
	state.UsePrefilledMinitia = true
	state.botConfig["da_node.bech32_prefix"] = "celestia"
	clone := state.Clone()
	require.True(t, clone.daIsCelestia)
	require.True(t, clone.UsePrefilledMinitia)
	clone.botConfig["da_node.bech32_prefix"] = "init"
	require.Equal(t, "celestia", state.botConfig["da_node.bech32_prefix"])
	state.InitExecutorBot = true
	state.MinitiaConfig = daArtifact()
	prefill, err := NewPrefillMinitiaConfig(daTestContext(t, state, t.TempDir()))
	require.NoError(t, err)
	next, _ := prefill.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fields := next.(*FieldInputModel)
	prefilling := weavecontext.GetCurrentState[OPInitBotsState](fields.Ctx)
	require.True(t, prefilling.UsePrefilledMinitia)
	require.False(t, prefilling.daIsCelestia)
	next, _ = fields.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	require.Same(t, prefill, next)
	restored := weavecontext.GetCurrentState[OPInitBotsState](prefill.Ctx)
	require.True(t, restored.daIsCelestia)
	require.True(t, restored.UsePrefilledMinitia)
	require.Equal(t, "celestia", restored.botConfig["da_node.bech32_prefix"])
	prefill.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, _ = prefill.Update(tea.KeyMsg{Type: tea.KeyEnter})
	l1 := next.(*L1PrefillSelector)
	next, _ = l1.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fields = next.(*FieldInputModel)
	next = daCompleteFields(t, fields)
	selector := next.(*SetDALayer)
	next, _ = selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
	loading := next.(*FetchL1StartHeightLoading)
	got := weavecontext.GetCurrentState[OPInitBotsState](loading.Ctx)
	require.False(t, got.daIsCelestia)
	require.False(t, got.UsePrefilledMinitia)
	require.Equal(t, "init", got.botConfig["da_node.bech32_prefix"])
}
