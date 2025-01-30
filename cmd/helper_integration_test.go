//go:build integration
// +build integration

package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

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

func getServiceFilePathAndBackupFilePath(serviceName service.CommandName) (string, string) {
	s, err := service.NewService(service.UpgradableInitia)
	if err != nil {
		panic(fmt.Errorf("failed to create service: %v", err))
	}

	serviceFilePath, err := s.GetServiceFile()
	if err != nil {
		panic(fmt.Errorf("failed to get service file: %v", err))
	}

	backupServiceFilePath := serviceFilePath + ".backup"

	return serviceFilePath, backupServiceFilePath
}

func backupServiceFiles(services []service.CommandName) {
	for _, serviceName := range services {
		serviceFilePath, backupServiceFilePath := getServiceFilePathAndBackupFilePath(serviceName)

		if _, err := os.Stat(serviceFilePath); os.IsNotExist(err) {
			continue
		}

		fmt.Printf("Backing up service file %s to %s\n", serviceFilePath, backupServiceFilePath)

		if err := os.Rename(serviceFilePath, backupServiceFilePath); err != nil {
			panic(fmt.Errorf("failed to backup service file: %v", err))
		}
	}
}

func restoreServiceFiles(services []service.CommandName) {
	for _, serviceName := range services {
		serviceFilePath, backupServiceFilePath := getServiceFilePathAndBackupFilePath(serviceName)

		if _, err := os.Stat(backupServiceFilePath); os.IsNotExist(err) {
			// remove the service file if the backup file does not exist
			os.Remove(serviceFilePath)
			continue
		}

		if err := os.Rename(backupServiceFilePath, serviceFilePath); err != nil {
			panic(fmt.Errorf("failed to restore service file: %v", err))
		}
	}
}

func setup(services []service.CommandName) {
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
			panic(fmt.Errorf("failed to backup weave directory: %v", err))
		}
	}

	if slices.Contains(services, service.Relayer) {
		relayerDir := filepath.Join(userHome, hermesDirectory)
		relayerDirBackup := filepath.Join(userHome, hermesDirectoryBackup)
		if _, err := os.Stat(relayerDir); !os.IsNotExist(err) {
			// remove the backup directory if it exists
			os.RemoveAll(relayerDirBackup)
			// rename the hermes directory to back up
			fmt.Println("Backing up hermes directory")

			if err := os.Rename(relayerDir, relayerDirBackup); err != nil {
				panic(fmt.Errorf("failed to backup hermes directory: %v", err))
			}
		}
	}

	// move service files to backup
	backupServiceFiles(services)
}

func teardown(services []service.CommandName) {
	userHome, _ := os.UserHomeDir()
	weaveDir := filepath.Join(userHome, common.WeaveDirectory)
	weaveDirBackup := filepath.Join(userHome, weaveDirectoryBackup)
	if _, err := os.Stat(weaveDirBackup); !os.IsNotExist(err) {
		fmt.Println("Restoring weave directory")
		os.RemoveAll(weaveDir)
		os.Rename(weaveDirBackup, weaveDir)
	}

	if slices.Contains(services, service.Relayer) {
		relayerDir := filepath.Join(userHome, hermesDirectory)
		relayerDirBackup := filepath.Join(userHome, hermesDirectoryBackup)
		if _, err := os.Stat(relayerDirBackup); !os.IsNotExist(err) {
			fmt.Println("Restoring hermes directory")
			os.RemoveAll(relayerDir)
			os.Rename(relayerDirBackup, relayerDir)
		}
	}

	// restore service files
	restoreServiceFiles(services)
}
