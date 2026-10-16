package cmd

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/internal/features"
	"github.com/stretchr/testify/require"
)

type opinitRoundTripFunc func(*http.Request) (*http.Response, error)

func (f opinitRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type opinitSnapshotFile struct {
	Mode  fs.FileMode
	Bytes string
}

func opinitSnapshot(t *testing.T, root string) map[string]opinitSnapshotFile {
	t.Helper()
	files := make(map[string]opinitSnapshotFile)
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		value := opinitSnapshotFile{Mode: info.Mode()}
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value.Bytes = string(data)
		}
		files[rel] = value
		return nil
	}))
	return files
}

// Catches preflight after keyfile generation/read or --force DB deletion.
func TestDAPolicyOPInitForcePreservesFiles(t *testing.T) {
	for _, generate := range []bool{false, true} {
		name := "key-file"
		if generate {
			name = "generate-key-file"
		}
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			oldAnalytics, oldProperties := analytics.Client, analytics.GlobalEventProperties
			analytics.Client = &analytics.NoOpClient{}
			analytics.GlobalEventProperties = nil
			t.Cleanup(func() {
				analytics.Client, analytics.GlobalEventProperties = oldAnalytics, oldProperties
			})
			opinitHome := filepath.Join(home, "opinit")
			paths := map[string]string{
				filepath.Join(opinitHome, "executor.db", "sentinel"):                                                  "database bytes",
				filepath.Join(opinitHome, "weave-dummy", "keyring", "key"):                                            "keyring bytes",
				filepath.Join(home, common.WeaveDataDirectory, common.OpinitGeneratedKeyFilename+".executor.keyfile"): "existing generated keyfile",
				filepath.Join(home, "input.keyfile"):                                                                  "{}",
				filepath.Join(home, "executor.json"):                                                                  `{"da_node":{"bech32_prefix":"celestia"},"disable_batch_submitter":true,"unknown":"preserve"}`,
			}
			for path, body := range paths {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
				require.NoError(t, os.WriteFile(path, []byte(body), 0640))
			}
			requests := 0
			old := http.DefaultTransport
			http.DefaultTransport = opinitRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests++
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: r}, nil
			})
			t.Cleanup(func() { http.DefaultTransport = old })
			before := opinitSnapshot(t, home)
			command := OPInitBotsInitCommand()
			flags := []string{"--with-config", filepath.Join(home, "executor.json"), "--" + FlagOPInitHome, opinitHome, "--force"}
			if generate {
				flags = append(flags, "--generate-key-file")
			} else {
				flags = append(flags, "--key-file", filepath.Join(home, "input.keyfile"))
			}
			require.NoError(t, command.ParseFlags(flags))
			err := command.RunE(command, []string{"executor"})
			if features.CelestiaDAEnabled {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no releases found")
				require.Equal(t, 1, requests)
			} else {
				require.True(t, errors.Is(err, common.ErrCelestiaDADisabled), "policy error: %v", err)
				require.Zero(t, requests, "executor validation must precede binary discovery")
				require.Equal(t, before, opinitSnapshot(t, home))
			}
		})
	}
}

// An unreadable keyfile cannot hide an unsupported executor config.
func TestDAPolicyOPInitConfigBeforeKeyfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, "executor.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"da_node":{"bech32_prefix":"unknown"}}`), 0600))
	err := handleWithConfig(OPInitBotsInitCommand(), home, filepath.Join(home, "opinit"), configPath, filepath.Join(home, "missing.keyfile"), []string{"executor"}, true, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported DA prefix")
}

// Catches decoding Challenger input as an executor and rejecting its absent DA.
func TestDAPolicyOPInitChallengerConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath, keyPath := filepath.Join(home, "challenger.json"), filepath.Join(home, "input.keyfile")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"l1_node":{"chain_id":"l1"},"l2_node":{"chain_id":"l2"}}`), 0600))
	require.NoError(t, os.WriteFile(keyPath, []byte(`{}`), 0600))
	requests := 0
	old := http.DefaultTransport
	http.DefaultTransport = opinitRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = old })
	err := handleWithConfig(OPInitBotsInitCommand(), home, filepath.Join(home, "opinit"), configPath, keyPath, []string{"challenger"}, false, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no releases found")
	require.Equal(t, 1, requests)
}
