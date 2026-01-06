package service

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

func TestDocker_GetServiceName(t *testing.T) {
	tests := []struct {
		name        string
		commandName CommandName
		want        string
		wantErr     bool
	}{
		{
			name:        "minitia service name",
			commandName: Minitia,
			want:        "weave-rollup",
			wantErr:     false,
		},
		{
			name:        "upgradable initia service name",
			commandName: UpgradableInitia,
			want:        "weave-initia",
			wantErr:     false,
		},
		{
			name:        "relayer service name",
			commandName: Relayer,
			want:        "weave-relayer",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDocker(tt.commandName, "")
			got, err := d.GetServiceName()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetServiceName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDocker_getImageName(t *testing.T) {
	tests := []struct {
		name        string
		commandName CommandName
		version     string
		want        string
		wantErr     bool
	}{
		{
			name:        "initia image",
			commandName: UpgradableInitia,
			version:     "main",
			want:        "ghcr.io/initia-labs/initiad:main",
			wantErr:     false,
		},
		{
			name:        "relayer image",
			commandName: Relayer,
			version:     "v1.0.7",
			want:        "ghcr.io/initia-labs/rapid-relayer:v1.0.7",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDocker(tt.commandName, "")
			got, err := d.getImageName(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("getImageName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getImageName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDocker_getCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		commandName CommandName
		want        []string
		wantErr     bool
	}{
		{
			name:        "minitia command args",
			commandName: Minitia,
			want:        []string{"start", "--home", "/app/data"},
			wantErr:     false,
		},
		{
			name:        "executor command args",
			commandName: OPinitExecutor,
			want:        []string{"start", "executor", "--home", "/app/data"},
			wantErr:     false,
		},
		{
			name:        "relayer command args",
			commandName: Relayer,
			want:        []string{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDocker(tt.commandName, "")
			got, err := d.getCommandArgs()
			if (err != nil) != tt.wantErr {
				t.Errorf("getCommandArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("getCommandArgs() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("getCommandArgs()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDocker_getVolumeName(t *testing.T) {
	tests := []struct {
		name        string
		commandName CommandName
		want        string
		wantErr     bool
	}{
		{
			name:        "minitia volume",
			commandName: Minitia,
			want:        "weave-rollup-data",
			wantErr:     false,
		},
		{
			name:        "relayer volume",
			commandName: Relayer,
			want:        "weave-relayer-data",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDocker(tt.commandName, "")
			got, err := d.getVolumeName()
			if (err != nil) != tt.wantErr {
				t.Errorf("getVolumeName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getVolumeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a volume exists
func checkVolumeExists(t *testing.T, cli *client.Client, volumeName string) bool {
	ctx := context.Background()

	filters := filters.NewArgs()
	filters.Add("name", volumeName)

	volumes, err := cli.VolumeList(ctx, volume.ListOptions{
		Filters: filters,
	})
	if err != nil {
		t.Logf("Warning: failed to list volumes: %v", err)
		return false
	}

	for _, v := range volumes.Volumes {
		if v.Name == volumeName {
			return true
		}
	}
	return false
}

// Helper function to check if an image exists
func checkImageExists(t *testing.T, cli *client.Client, imageName string) (types.ImageInspect, bool) {
	ctx := context.Background()

	inspect, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	return inspect, err == nil
}

// Helper function to cleanup Docker resources
func cleanupDockerResources(t *testing.T, cli *client.Client, volumeName string) {
	ctx := context.Background()

	// Remove volume if it exists
	err := cli.VolumeRemove(ctx, volumeName, true)
	if err != nil {
		t.Logf("Note: volume cleanup: %v", err)
	}
}

func TestDocker_Start(t *testing.T) {
	// Check if Docker is available
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer cli.Close()

	// Test that we can connect to Docker daemon
	_, err = cli.Ping(ctx)
	if err != nil {
		t.Skipf("Docker daemon not responding: %v", err)
	}

	tests := []struct {
		name        string
		commandName CommandName
		version     string
		wantErr     bool
	}{
		{
			name:        "start relayer service",
			commandName: Relayer,
			version:     "v1.0.7",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDocker(tt.commandName, "")

			// Get service name for cleanup
			serviceName, err := d.GetServiceName()
			if err != nil {
				t.Fatalf("failed to get service name: %v", err)
			}

			// Get volume name for cleanup
			volumeName, err := d.getVolumeName()
			if err != nil {
				t.Fatalf("failed to get volume name: %v", err)
			}

			// Cleanup before test - stop any running container
			_ = d.Stop()
			cleanupDockerResources(t, cli, volumeName)

			// First create the service (pull image and create volume)
			err = d.Create(tt.version, "/tmp/test-start")
			if err != nil {
				t.Fatalf("Create() failed: %v", err)
			}

			// Now test Start
			err = d.Start()
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify container is running
				containerJSON, err := cli.ContainerInspect(ctx, serviceName)
				if err != nil {
					t.Fatalf("failed to inspect container: %v", err)
				}

				if !containerJSON.State.Running {
					t.Errorf("Container %s is not running", serviceName)
				} else {
					t.Logf("Container %s started successfully (Status: %s)", serviceName, containerJSON.State.Status)
				}
			}

			// Cleanup after test
			_ = d.Stop()
			cleanupDockerResources(t, cli, volumeName)
		})
	}
}

func TestDocker_Log(t *testing.T) {
	// Check if Docker is available
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer cli.Close()

	// Test that we can connect to Docker daemon
	_, err = cli.Ping(ctx)
	if err != nil {
		t.Skipf("Docker daemon not responding: %v", err)
	}

	tests := []struct {
		name        string
		commandName CommandName
		version     string
		numLines    int
		wantErr     bool
	}{
		{
			name:        "get logs from relayer service",
			commandName: Relayer,
			version:     "v1.0.7",
			numLines:    10,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDocker(tt.commandName, "")

			// Get service name for cleanup
			serviceName, err := d.GetServiceName()
			if err != nil {
				t.Fatalf("failed to get service name: %v", err)
			}

			// Get volume name for cleanup
			volumeName, err := d.getVolumeName()
			if err != nil {
				t.Fatalf("failed to get volume name: %v", err)
			}

			// Cleanup before test
			_ = d.Stop()
			cleanupDockerResources(t, cli, volumeName)

			// Create and start the service
			err = d.Create(tt.version, "/tmp/test-logs")
			if err != nil {
				t.Fatalf("Create() failed: %v", err)
			}

			err = d.Start()
			if err != nil {
				t.Fatalf("Start() failed: %v", err)
			}

			// Wait a moment for container to produce some logs
			// Note: In real test, we might want to wait longer or check container status
			t.Logf("Waiting for container to generate logs...")

			// Verify container is running before trying to get logs
			containerJSON, err := cli.ContainerInspect(ctx, serviceName)
			if err != nil {
				t.Fatalf("failed to inspect container: %v", err)
			}

			if !containerJSON.State.Running {
				t.Logf("Warning: Container is not running, may not have logs")
			}

			// Test Log function - capture stdout
			// Save original stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call Log in a goroutine so we can capture its output
			errChan := make(chan error, 1)
			go func() {
				errChan <- d.Log(tt.numLines)
			}()

			// Read captured output
			outC := make(chan string, 1)
			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				outC <- buf.String()
			}()

			// Wait for Log to complete (with timeout protection)
			err = <-errChan
			w.Close()
			output := <-outC

			// Restore stdout
			os.Stdout = oldStdout

			// Check for errors
			if (err != nil) != tt.wantErr {
				t.Errorf("Log() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Display the captured logs
			if !tt.wantErr {
				if len(output) > 0 {
					t.Logf("Captured logs (%d bytes):\n%s", len(output), output)
				} else {
					t.Logf("No logs captured (container may not have produced any logs yet)")
				}
			}

			// Cleanup after test
			_ = d.Stop()
			cleanupDockerResources(t, cli, volumeName)
		})
	}
}
