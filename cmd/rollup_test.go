package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/internal/features"
	"github.com/test-go/testify/require"
)

// Catches a missing command preflight before --force can remove a rollup home.
func TestDAPolicyRollupForcePreservesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rollupHome := filepath.Join(home, "rollup")
	require.NoError(t, os.MkdirAll(rollupHome, 0700))
	sentinel := filepath.Join(rollupHome, "sentinel")
	require.NoError(t, os.WriteFile(sentinel, []byte("preserve"), 0600))
	cfgPath := filepath.Join(home, "config.json")
	cfg := []byte(`{"op_bridge":{"batch_submission_target":"CELESTIA"},"unknown_field":"preserve"}`)
	require.NoError(t, os.WriteFile(cfgPath, cfg, 0600))
	cmd := minitiaLaunchCommand()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--with-config", cfgPath, "--vm", "move", "--force", "--" + FlagMinitiaHome, rollupHome}))
	err := cmd.PreRunE(cmd, nil)
	if features.CelestiaDAEnabled {
		require.NoError(t, err)
	} else {
		require.True(t, errors.Is(err, common.ErrCelestiaDADisabled), "policy error: %v", err)
	}
	actual, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	require.Equal(t, "preserve", string(actual))
	actual, err = os.ReadFile(cfgPath)
	require.NoError(t, err)
	require.Equal(t, cfg, actual)
}
