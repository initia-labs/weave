//go:build integration
// +build integration

package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/service"
)

const (
	weaveDirectoryBackup  = ".weave_backup"
	hermesDirectory       = ".hermes"
	hermesDirectoryBackup = ".hermes_backup"

	TestMinitiaHome = ".minitia.weave.test"
	TestOPInitHome  = ".opinit.weave.test"
	TestInitiaHome  = ".initia.weave.test"
)

func getServiceNames(services []service.Command) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, string(service.Name))
	}
	return names
}

func setup(t *testing.T, services []service.Command) {
	// disable analytics
	analytics.Client = &analytics.NoOpClient{}

	userHome, _ := os.UserHomeDir()
	weaveDir := filepath.Join(userHome, common.WeaveDirectory)
	weaveDirBackup := filepath.Join(userHome, weaveDirectoryBackup)
	if _, err := os.Stat(weaveDir); !os.IsNotExist(err) {
		// remove the backup directory if it exists
		os.RemoveAll(weaveDirBackup)
		// rename the weave directory to back up
		fmt.Println("Backing up weave directory")

		if err := os.Rename(weaveDir, weaveDirBackup); err != nil {
			t.Fatalf("failed to backup weave directory: %v", err)
		}
	}

	if slices.Contains(getServiceNames(services), string(service.Relayer.Name)) {
		relayerDir := filepath.Join(userHome, hermesDirectory)
		relayerDirBackup := filepath.Join(userHome, hermesDirectoryBackup)
		if _, err := os.Stat(relayerDir); !os.IsNotExist(err) {
			// remove the backup directory if it exists
			os.RemoveAll(relayerDirBackup)
			// rename the hermes directory to back up
			fmt.Println("Backing up hermes directory")

			if err := os.Rename(relayerDir, relayerDirBackup); err != nil {
				t.Fatalf("failed to backup hermes directory: %v", err)
			}
		}
	}
}

func teardown(t *testing.T, services []service.Command) {
	userHome, _ := os.UserHomeDir()
	weaveDir := filepath.Join(userHome, common.WeaveDirectory)
	weaveDirBackup := filepath.Join(userHome, weaveDirectoryBackup)
	if _, err := os.Stat(weaveDirBackup); !os.IsNotExist(err) {
		fmt.Println("Restoring weave directory")
		os.RemoveAll(weaveDir)
		os.Rename(weaveDirBackup, weaveDir)
	}

	if slices.Contains(getServiceNames(services), string(service.Relayer.Name)) {
		relayerDir := filepath.Join(userHome, hermesDirectory)
		relayerDirBackup := filepath.Join(userHome, hermesDirectoryBackup)
		if _, err := os.Stat(relayerDirBackup); !os.IsNotExist(err) {
			fmt.Println("Restoring hermes directory")
			os.RemoveAll(relayerDir)
			os.Rename(relayerDirBackup, relayerDir)
		}
	}
}
