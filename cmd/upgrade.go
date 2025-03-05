package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fynelabs/selfupdate"
	"github.com/spf13/cobra"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/io"
)

const (
	WeaveReleaseAPI string = "https://api.github.com/repos/initia-labs/weave/releases"
	WeaveReleaseURL string = "https://www.github.com/initia-labs/weave/releases"
)

func VersionCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the Weave binary version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), Version)
			if err != nil {
				return err
			}
			return nil
		},
	}

	return versionCmd
}

func UpgradeCommand() *cobra.Command {
	upgradeCmd := &cobra.Command{
		Use:   "upgrade [version]",
		Short: "Upgrade the Weave binary to the latest or a specified version from GitHub",
		Long: `Upgrade the Weave binary to the latest available release from GitHub or a specified version.

Examples:
  weave upgrade            Upgrade to the latest release
  weave upgrade 1.2.3      Upgrade to a specific version (v1.2.3)
  weave upgrade 1.2        Upgrade to the latest patch version of 1.2
  weave upgrade 1          Upgrade to the latest minor version of 1.x

If the specified version does not exist, an error will be shown with a link to the available releases.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requestedVersion := ""
			if len(args) > 0 {
				requestedVersion = args[0]
				if requestedVersion == Version {
					fmt.Printf("ℹ️ The current Weave version matches the specified version.\n\n")
					return nil
				}
				isNewer := cosmosutils.CompareSemVer(requestedVersion, Version)
				if !isNewer {
					return fmt.Errorf("the specified version is older than the current version: %s", Version)
				}
				return handleUpgrade(requestedVersion)
			}
			requestedVersion, _, err := cosmosutils.GetLatestWeaveVersion()
			if err != nil {
				return fmt.Errorf("failed to get latest weave version: %w", err)
			}
			isNewer := cosmosutils.CompareSemVer(requestedVersion, Version)
			if !isNewer {
				return fmt.Errorf("the specified version is older than the current version: %s", Version)
			}
			return handleUpgrade(requestedVersion)
		},
	}

	return upgradeCmd
}

func handleUpgrade(requestedVersion string) error {
	availableVersions, err := cosmosutils.ListWeaveReleases(WeaveReleaseAPI)
	if err != nil {
		return err
	}
	if len(availableVersions) == 0 {
		return fmt.Errorf("failed to fetch available Weave versions")
	}

	fmt.Printf("⚙️ Upgrading to version %s...\n", requestedVersion)
	downloadURL := availableVersions[requestedVersion]
	if err := downloadAndReplaceBinary(downloadURL); err != nil {
		return fmt.Errorf("failed to upgrade to version %s: %w", requestedVersion, err)
	}

	fmt.Printf("✅ Upgrade successful! You are now using %s of Weave.\n\n", requestedVersion)
	return nil
}

func downloadAndReplaceBinary(downloadURL string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	tarballPath := filepath.Join(homeDir, common.WeaveDataDirectory, "weave-binary.tar.gz")
	extractedPath := filepath.Join(homeDir, common.WeaveDataDirectory)
	binaryPath := filepath.Join(extractedPath, "weave")
	fmt.Printf("⬇️ Downloading from %s...\n", downloadURL)

	if err = io.DownloadAndExtractTarGz(downloadURL, tarballPath, extractedPath); err != nil {
		return fmt.Errorf("failed to download and extract binary: %v", err)
	}
	defer func() {
		_ = io.DeleteFile(binaryPath)
	}()

	if err = doReplace(binaryPath); err != nil {
		return fmt.Errorf("failed to replace the new weave binary: %v", err)
	}

	return nil
}

func doReplace(binaryPath string) error {
	newBinary, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded binary: %w", err)
	}
	defer func() {
		_ = newBinary.Close()
	}()

	err = selfupdate.Apply(newBinary, selfupdate.Options{})
	if err != nil {
		if rollbackErr := selfupdate.RollbackError(err); rollbackErr != nil {
			return fmt.Errorf("failed to apply update and rollback failed: %v", rollbackErr)
		}
		return fmt.Errorf("failed to apply update: %w", err)
	}

	return nil
}
