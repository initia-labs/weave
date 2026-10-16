//go:build e2e

package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/test-go/testify/require"
)

const e2eDisabledPolicy = "Celestia DA is disabled in this build; use Initia DA or a Celestia-enabled build"

func buildWeaveBinary(t *testing.T, celestia bool) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "weave")
	tags := ""
	if celestia {
		tags = "celestia"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binary, "-tags="+tags, ".")
	cmd.Dir = ".."
	// The outer e2e/celestia test tags must not leak into either main binary.
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "GOFLAGS=") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "GOFLAGS=")
	boundWeaveProcess(cmd)
	output, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "binary build timed out: %s", output)
	require.NoError(t, err, "build tags=%q: %s", tags, output)
	return binary
}

// Kill the whole child process group on timeout, and always wait/reap via
// CombinedOutput. No daemon is started by the early-boundary cases below.
func boundWeaveProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 5 * time.Second
}

func runWeave(t *testing.T, binary string, args []string, env []string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	boundWeaveProcess(cmd)
	output, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "CLI timed out for %v: %s", args, output)
	if err == nil {
		return 0, string(output)
	}
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr), "failed to execute CLI: %v", err)
	return exitErr.ExitCode(), string(output)
}

func weaveE2EHome(t *testing.T) (string, []string) {
	t.Helper()
	home := t.TempDir()
	for _, dir := range []string{".weave/data", ".weave/log", "bin"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, dir), 0700))
	}
	lifecycleWrite(t, filepath.Join(home, ".weave/config.json"), `{"common":{"analytics_opt_out":true,"analytics_device_id":"task6-fixed-device"}}`, 0600)
	// Root only uses LookPath. Execution would be outside the tested boundary.
	lifecycleWrite(t, filepath.Join(home, "bin/lz4"), "#!/bin/sh\nprintf called > \"$HOME/lz4-executed\"\nexit 99\n", 0700)
	env := []string{}
	for _, value := range os.Environ() {
		key := strings.SplitN(value, "=", 2)[0]
		if key == "HOME" || key == "PATH" || strings.HasPrefix(key, "WEAVE_") {
			continue
		}
		env = append(env, value)
	}
	env = append(env, "HOME="+home, "PATH="+filepath.Join(home, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return home, env
}

// Executes real main binaries and catches policy checks moved after --force
// deletes/key generation, build-tag contamination, and Initia/Challenger overblocking.
func TestCelestiaPolicyE2E(t *testing.T) {
	for _, celestia := range []bool{false, true} {
		mode := "default"
		if celestia {
			mode = "celestia"
		}
		t.Run(mode, func(t *testing.T) {
			binary := buildWeaveBinary(t, celestia)
			cases := []struct {
				name, kind, fixture, body, expected string
				force, generate, defaultOnly        bool
			}{
				{name: "celestia rollup existing", kind: "rollup", fixture: "rollup-celestia.json", expected: "rollup-conflict"},
				{name: "initia rollup existing", kind: "rollup", fixture: "rollup-initia.json", expected: "rollup-conflict"},
				{name: "celestia rollup force", kind: "rollup", fixture: "rollup-celestia.json", force: true, defaultOnly: true, expected: "policy"},
				{name: "celestia executor missing key force", kind: "executor", fixture: "executor-celestia.json", force: true, expected: "missing-key"},
				{name: "initia executor missing key force", kind: "executor", fixture: "executor-initia.json", force: true, expected: "missing-key"},
				{name: "celestia executor generate force", kind: "executor", fixture: "executor-celestia.json", force: true, generate: true, defaultOnly: true, expected: "policy"},
				{name: "challenger missing key", kind: "challenger", fixture: "challenger.json", expected: "missing-key"},
				{name: "rollup empty target", kind: "rollup", body: `{"op_bridge":{"batch_submission_target":""}}`, expected: `unsupported DA target "": expected INITIA or CELESTIA`},
				{name: "rollup unknown target", kind: "rollup", body: `{"op_bridge":{"batch_submission_target":"UNKNOWN"}}`, expected: `unsupported DA target "UNKNOWN": expected INITIA or CELESTIA`},
				{name: "executor empty prefix", kind: "executor", body: `{"da_node":{"bech32_prefix":""}}`, expected: `unsupported DA prefix "": expected init or celestia`},
				{name: "executor unknown prefix", kind: "executor", body: `{"da_node":{"bech32_prefix":"unknown"}}`, expected: `unsupported DA prefix "unknown": expected init or celestia`},
			}
			for _, tc := range cases {
				if celestia && tc.defaultOnly {
					continue // tagged --force/generate would cross into real launch/key work
				}
				t.Run(tc.name, func(t *testing.T) {
					home, env := weaveE2EHome(t)
					protected := filepath.Join(home, "protected")
					rollupHome, opinitHome := filepath.Join(protected, "rollup"), filepath.Join(protected, "opinit")
					lifecycleWrite(t, filepath.Join(rollupHome, "sentinel"), "keep rollup", 0640)
					lifecycleWrite(t, filepath.Join(opinitHome, "executor.db", "sentinel"), "keep database", 0600)
					lifecycleWrite(t, filepath.Join(opinitHome, "keyring-test", "sentinel"), "keep keys", 0600)
					lifecycleWrite(t, filepath.Join(protected, "service.definition"), "keep service", 0640)
					require.NoError(t, os.Mkdir(filepath.Join(protected, "empty-dir"), 0750))
					require.NoError(t, os.Symlink("executor.db/sentinel", filepath.Join(opinitHome, "db-link")))
					// Keep a generated-key sentinel outside protected as the real command
					// writes this exact root-bootstrap data path when generation is allowed.
					generated := filepath.Join(home, ".weave/data/weave.opinit.generated.executor.keyfile")
					lifecycleWrite(t, generated, "existing generated keyfile", 0640)
					body := tc.body
					if tc.fixture != "" {
						data, err := os.ReadFile(filepath.Join("testdata/da", tc.fixture))
						require.NoError(t, err)
						body = string(data)
					}
					configPath := filepath.Join(protected, "input.json")
					lifecycleWrite(t, configPath, body, 0640)
					missing := filepath.Join(protected, "missing.keyfile")
					_, err := os.Lstat(missing)
					require.True(t, os.IsNotExist(err))
					args := []string{"rollup", "launch", "--with-config", configPath, "--vm", "move", "--minitia-dir", rollupHome}
					if tc.kind != "rollup" {
						args = []string{"opinit", "init", tc.kind, "--with-config", configPath, "--opinit-dir", opinitHome}
						if tc.generate {
							args = append(args, "--generate-key-file")
						} else {
							args = append(args, "--key-file", missing)
						}
					}
					if tc.force {
						args = append(args, "--force")
					}
					before := daProtectedSnapshot(t, protected)
					generatedBefore := daProtectedSnapshot(t, filepath.Dir(generated))
					want := tc.expected
					if !celestia && (tc.fixture == "rollup-celestia.json" || tc.fixture == "executor-celestia.json") {
						want = "policy"
					}
					switch want {
					case "policy":
						if tc.kind == "rollup" {
							want = `DA target "CELESTIA": ` + e2eDisabledPolicy
						} else {
							want = `DA prefix "celestia": ` + e2eDisabledPolicy
						}
					case "rollup-conflict":
						want = "existing " + rollupHome + " folder detected. Use --force or -f to override"
					case "missing-key":
						want = "key file is missing at path: " + missing
					}
					exitCode, output := runWeave(t, binary, args, env)
					require.Equal(t, 1, exitCode, "output: %s", output)
					require.Contains(t, output, want)
					if !strings.Contains(want, e2eDisabledPolicy) {
						require.NotContains(t, output, e2eDisabledPolicy)
					}
					require.NotContains(t, output, "panic:")
					require.Equal(t, before, daProtectedSnapshot(t, protected))
					require.Equal(t, generatedBefore, daProtectedSnapshot(t, filepath.Dir(generated)))
					_, err = os.Lstat(filepath.Join(home, "lz4-executed"))
					require.True(t, os.IsNotExist(err), "lz4 must only be looked up")
					t.Logf("exit=%d; expected error: %s; protected entries=%d unchanged", exitCode, want, len(before))
				})
			}
		})
	}
}
