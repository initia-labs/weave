package cmd

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/initia-labs/weave/service"
	"github.com/spf13/cobra"
	"github.com/test-go/testify/require"
)

type lifecycleService struct {
	name                      service.CommandName
	serviceFile, binary, home string
	calls                     *[]string
}

func (s *lifecycleService) record(method string) {
	*s.calls = append(*s.calls, string(s.name)+":"+method)
}
func (s *lifecycleService) Create(string, string) error { return fmt.Errorf("unexpected Create") }
func (s *lifecycleService) PruneLogs() error            { return fmt.Errorf("unexpected PruneLogs") }
func (s *lifecycleService) Start(args ...string) error {
	s.record(fmt.Sprintf("Start%v", args))
	return nil
}
func (s *lifecycleService) Stop() error    { s.record("Stop"); return nil }
func (s *lifecycleService) Restart() error { s.record("Restart"); return nil }
func (s *lifecycleService) Log(n int) error {
	s.record(fmt.Sprintf("Log(%d)", n))
	return nil
}
func (s *lifecycleService) GetServiceFile() (string, error) {
	s.record("GetServiceFile")
	return s.serviceFile, nil
}
func (s *lifecycleService) GetServiceBinaryAndHome() (string, string, error) {
	s.record("GetServiceBinaryAndHome")
	return s.binary, s.home, nil
}

func lifecycleWrite(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(body), mode))
}

type daProtectedEntry struct {
	Mode   os.FileMode
	Data   string
	Target string
}

// WalkDir/Lstat preserve symlink identity without reading its target, and include
// directory entries/modes while excluding volatile access times.
func daProtectedSnapshot(t *testing.T, root string) map[string]daProtectedEntry {
	t.Helper()
	entries := make(map[string]daProtectedEntry)
	require.NoError(t, filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		entry := daProtectedEntry{Mode: info.Mode()}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			entry.Target, err = os.Readlink(path)
		case info.Mode().IsRegular():
			var data []byte
			data, err = os.ReadFile(path)
			entry.Data = string(data)
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		entries[rel] = entry
		return err
	}))
	return entries
}

func lifecycleFixtures(t *testing.T) (map[service.CommandName]*lifecycleService, *[]string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	calls := []string{}
	services := make(map[service.CommandName]*lifecycleService)
	for _, name := range []service.CommandName{service.Minitia, service.OPinitExecutor, service.OPinitChallenger} {
		s := &lifecycleService{name: name, calls: &calls,
			serviceFile: filepath.Join(home, "services", string(name)+".service"),
			binary:      filepath.Join(home, "bin", string(name)), home: filepath.Join(home, string(name))}
		lifecycleWrite(t, s.serviceFile, "existing service definition", 0640)
		lifecycleWrite(t, s.binary, "existing binary; must not execute", 0700)
		lifecycleWrite(t, filepath.Join(s.home, "sentinel"), "existing data", 0600)
		services[name] = s
	}
	rollupArtifact, err := os.ReadFile("testdata/da/rollup-celestia.json")
	require.NoError(t, err)
	lifecycleWrite(t, filepath.Join(services[service.Minitia].home, "artifacts", "config.json"), string(rollupArtifact), 0640)
	executorConfig, err := os.ReadFile("testdata/da/executor-celestia.json")
	require.NoError(t, err)
	lifecycleWrite(t, filepath.Join(services[service.OPinitExecutor].home, "executor.json"), string(executorConfig), 0600)
	lifecycleWrite(t, filepath.Join(services[service.OPinitChallenger].home, "challenger.json"), `{}`, 0600)
	old := newService
	newService = func(name service.CommandName, vm string) (service.Service, error) {
		require.Empty(t, vm)
		calls = append(calls, string(name)+":factory")
		s, ok := services[name]
		require.True(t, ok, "unexpected service %q", name)
		return s, nil
	}
	t.Cleanup(func() { newService = old })
	return services, &calls
}

func lifecycleRoot() *cobra.Command {
	root := &cobra.Command{Use: "weave", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(MinitiaCommand(), OPInitBotsCommand())
	return root
}

// Catches policy gates on existing services, wrong bot dispatch, lost polling/log
// arguments, and any attempt to rewrite configs or access DA over HTTP.
func TestServiceLifecyclePreservesCelestia(t *testing.T) {
	for _, target := range []string{"rollup", "executor", "challenger"} {
		for _, verb := range []string{"start", "stop", "restart", "log"} {
			t.Run(target+"/"+verb, func(t *testing.T) {
				services, calls := lifecycleFixtures(t)
				home := filepath.Dir(services[service.Minitia].home)
				before := daProtectedSnapshot(t, home)
				requests := []string{}
				oldTransport := http.DefaultTransport
				http.DefaultTransport = opinitRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					requests = append(requests, r.URL.String())
					return nil, fmt.Errorf("unexpected lifecycle HTTP request %s", r.URL)
				})
				t.Cleanup(func() { http.DefaultTransport = oldTransport })
				name, preflight := service.Minitia, service.Minitia
				args := []string{"rollup", verb}
				if target != "rollup" {
					name, preflight = service.CommandName(target), service.OPinitExecutor
					args = []string{"opinit", verb, target}
				}
				method := map[string]string{"start": "Start[]", "stop": "Stop", "restart": "Restart", "log": "Log(7)"}[verb]
				if verb == "start" {
					args = append(args, "--detach")
					if target != "rollup" {
						args = append(args, "--polling-interval", "250ms")
						method = "Start[--polling-interval=250ms]"
					}
				} else if verb == "log" {
					args = append(args, "-n", "7")
				}
				root := lifecycleRoot()
				root.SetArgs(args)
				require.NoError(t, root.Execute())
				want := []string{}
				if verb != "log" {
					// Baseline OPinit preflight always validates executor, even for challenger.
					want = append(want, string(preflight)+":factory", string(preflight)+":GetServiceFile", string(preflight)+":GetServiceBinaryAndHome")
				}
				want = append(want, string(name)+":factory", string(name)+":"+method)
				require.Equal(t, want, *calls)
				require.Empty(t, requests)
				require.Equal(t, before, daProtectedSnapshot(t, home))
			})
		}
	}
}

// Catches a seam that accidentally bypasses service file/home/binary validation.
func TestServiceLifecycleRequiresExistingPaths(t *testing.T) {
	for _, missing := range []string{"service", "home", "binary"} {
		t.Run(missing, func(t *testing.T) {
			services, calls := lifecycleFixtures(t)
			s := services[service.OPinitExecutor]
			path := map[string]string{"service": s.serviceFile, "home": s.home, "binary": s.binary}[missing]
			require.NoError(t, os.RemoveAll(path))
			root := lifecycleRoot()
			root.SetArgs([]string{"opinit", "start", "executor", "--detach"})
			err := root.Execute()
			require.Error(t, err)
			require.Contains(t, err.Error(), "not properly configured")
			require.Contains(t, err.Error(), path)
			want := []string{"executor:factory", "executor:GetServiceFile"}
			if missing != "service" {
				want = append(want, "executor:GetServiceBinaryAndHome")
			}
			require.Equal(t, want, *calls)
		})
	}
}

func TestServiceLifecycleValidatesBotName(t *testing.T) {
	for _, verb := range []string{"start", "stop", "restart", "log"} {
		t.Run(verb, func(t *testing.T) {
			_, calls := lifecycleFixtures(t)
			root := lifecycleRoot()
			root.SetArgs([]string{"opinit", verb, "other"})
			err := root.Execute()
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid bot name 'other'")
			require.Empty(t, *calls)
		})
	}
}
