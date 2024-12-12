package cosmosutils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/io"
)

type BinaryRelease struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type BinaryVersionWithDownloadURL map[string]string

func getOSArch() (os, arch string) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "darwin":
		os = "Darwin"
	case "linux":
		os = "Linux"
	default:
		panic(fmt.Errorf("unsupported OS: %s", goos))
	}

	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	default:
		panic(fmt.Errorf("unsupported architecture: %s", goarch))
	}

	return os, arch
}

func fetchReleases(url string) []BinaryRelease {
	httpClient := client.NewHTTPClient()
	var releases []BinaryRelease
	_, err := httpClient.Get(url, "", nil, &releases)
	if err != nil {
		panic(fmt.Errorf("failed to fetch releases: %v", err))
	}

	return releases
}

func mapReleasesToVersions(releases []BinaryRelease) BinaryVersionWithDownloadURL {
	versions := make(BinaryVersionWithDownloadURL)
	goos, arch := getOSArch()
	searchString := fmt.Sprintf("%s_%s.tar.gz", goos, arch)

	for _, release := range releases {
		for _, asset := range release.Assets {
			if strings.Contains(asset.BrowserDownloadURL, searchString) {
				versions[release.TagName] = asset.BrowserDownloadURL
			}
		}
	}

	return versions
}

func ListBinaryReleases(url string) BinaryVersionWithDownloadURL {
	releases := fetchReleases(url)
	return mapReleasesToVersions(releases)
}

func GetLatestMinitiaVersion(vm string) (string, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/initia-labs/mini%s/releases", vm)
	releases := fetchReleases(url)

	if len(releases) < 1 {
		return "", "", fmt.Errorf("no releases found")
	}

	goos, arch := getOSArch()
	searchString := fmt.Sprintf("%s_%s.tar.gz", goos, arch)

	var latestRelease *BinaryRelease
	var downloadURL string

	for _, release := range releases {
		for _, asset := range release.Assets {
			if strings.Contains(asset.BrowserDownloadURL, searchString) {
				if latestRelease == nil || compareDates(latestRelease.PublishedAt, release.PublishedAt) {
					latestRelease = &release
					downloadURL = asset.BrowserDownloadURL
				}
				break
			}
		}
	}

	if latestRelease == nil {
		return "", "", fmt.Errorf("no compatible release found for %s_%s", goos, arch)
	}

	return latestRelease.TagName, downloadURL, nil
}

// SortVersions sorts the versions based on semantic versioning, including pre-release handling
func SortVersions(versions BinaryVersionWithDownloadURL) []string {
	var versionTags []string
	for tag := range versions {
		versionTags = append(versionTags, tag)
	}

	// Sort based on major, minor, patch, and pre-release metadata
	sort.Slice(versionTags, func(i, j int) bool {
		return CompareSemVer(versionTags[i], versionTags[j])
	})

	return versionTags
}

func compareDates(d1, d2 string) bool {
	const layout = time.RFC3339

	t1, err1 := time.Parse(layout, d1)
	t2, err2 := time.Parse(layout, d2)

	if err1 != nil && err2 != nil {
		return d1 < d2 // fallback
	} else if err1 != nil {
		return false
	} else if err2 != nil {
		return true
	}

	return t1.Before(t2)
}

// CompareSemVer compares two semantic version strings and returns true if v1 should be ordered before v2
func CompareSemVer(v1, v2 string) bool {
	// Trim "v" prefix
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	v1Main, v1Pre := splitVersion(v1)
	v2Main, v2Pre := splitVersion(v2)

	// Compare the main (major, minor, patch) versions
	v1MainParts := strings.Split(v1Main, ".")
	v2MainParts := strings.Split(v2Main, ".")
	for i := 0; i < 3; i++ {
		v1Part, _ := strconv.Atoi(v1MainParts[i])
		v2Part, _ := strconv.Atoi(v2MainParts[i])

		if v1Part != v2Part {
			return v1Part > v2Part
		}
	}

	// Compare pre-release parts if main versions are equal
	// A pre-release version is always ordered lower than the normal version
	if v1Pre == "" && v2Pre != "" {
		return true
	}
	if v1Pre != "" && v2Pre == "" {
		return false
	}
	return v1Pre > v2Pre
}

// splitVersion separates the main version (e.g., "0.4.11") from the pre-release (e.g., "Binarytion.1")
func splitVersion(version string) (mainVersion, preRelease string) {
	if strings.Contains(version, "-") {
		parts := strings.SplitN(version, "-", 2)
		return parts[0], parts[1]
	}
	return version, ""
}

func GetOPInitVersions() (BinaryVersionWithDownloadURL, string) {
	versions := ListBinaryReleases("https://api.github.com/repos/initia-labs/opinit-bots/releases")
	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	binaryPath := filepath.Join(userHome, common.WeaveDataDirectory, common.OPinitAppName)
	currentVersion, _ := GetBinaryVersion(binaryPath)

	return versions, currentVersion
}

func MustGetInitiaBinaryUrlFromLcd(httpClient *client.HTTPClient, rest string) (string, string) {
	var result NodeInfoResponse
	_, err := httpClient.Get(rest, "/cosmos/base/tendermint/v1beta1/node_info", nil, &result)
	if err != nil {
		panic(err)
	}

	version := result.ApplicationVersion.Version
	url := getBinaryURL(version)

	return version, url
}

func getBinaryURL(version string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "darwin":
		switch goarch {
		case "amd64":
			return fmt.Sprintf("https://github.com/initia-labs/initia/releases/download/%s/initia_%s_Darwin_x86_64.tar.gz", version, version)
		case "arm64":
			return fmt.Sprintf("https://github.com/initia-labs/initia/releases/download/%s/initia_%s_Darwin_aarch64.tar.gz", version, version)
		}
	case "linux":
		switch goarch {
		case "amd64":
			return fmt.Sprintf("https://github.com/initia-labs/initia/releases/download/%s/initia_%s_Linux_x86_64.tar.gz", version, version)
		case "arm64":
			return fmt.Sprintf("https://github.com/initia-labs/initia/releases/download/%s/initia_%s_Linux_aarch64.tar.gz", version, version)
		}
	}
	panic("unsupported OS or architecture")
}

func GetInitiaBinaryPath(version string) string {
	if strings.Contains(version, "@") {
		parts := strings.Split(version, "@")
		if len(parts) == 2 {
			version = parts[1]
		} else {
			panic(fmt.Sprintf("invalid version format: %s", version))
		}
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}
	extractedPath := filepath.Join(userHome, common.WeaveDataDirectory, fmt.Sprintf("initia@%s", version))

	switch runtime.GOOS {
	case "linux":
		if CompareSemVer(version, "v0.6.1") {
			return filepath.Join(extractedPath, "initiad")
		}
		return filepath.Join(extractedPath, "initia_"+version, "initiad")
	case "darwin":
		return filepath.Join(extractedPath, "initiad")
	default:
		panic("unsupported OS")
	}
}

func MustInstallInitiaBinary(version, url, binaryPath string) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}
	tarballPath := filepath.Join(userHome, common.WeaveDataDirectory, "initia.tar.gz")
	extractedPath := filepath.Join(userHome, common.WeaveDataDirectory, fmt.Sprintf("initia@%s", version))

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
			err := os.MkdirAll(extractedPath, os.ModePerm)
			if err != nil {
				panic(fmt.Sprintf("failed to create weave data directory: %v", err))
			}
		}

		if err = io.DownloadAndExtractTarGz(url, tarballPath, extractedPath); err != nil {
			panic(fmt.Sprintf("failed to download and extract binary: %v", err))
		}

		err = os.Chmod(binaryPath, 0755)
		if err != nil {
			panic(fmt.Sprintf("failed to set permissions for binary: %v", err))
		}
	}

	io.SetLibraryPaths(filepath.Dir(binaryPath))
}

func MustInstallCosmovisor(version string) string {
	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}
	targetDirectory := filepath.Join(userHome, common.WeaveDataDirectory)
	tarballPath := filepath.Join(userHome, common.WeaveDataDirectory, "cosmovisor.tar.gz")
	extractedPath := filepath.Join(targetDirectory, fmt.Sprintf("cosmovisor@%s", version))

	binaryPath := filepath.Join(userHome, common.WeaveDataDirectory, fmt.Sprintf("cosmovisor@%s", version), "cosmovisor")

	// Determine the OS and architecture
	osType := runtime.GOOS
	arch := runtime.GOARCH
	if osType == "darwin" {
		arch = "amd64"
	}

	// Generate the URL dynamically
	supportedCombinations := map[string]string{
		"linux-amd64":  "cosmovisor-%s-linux-amd64.tar.gz",
		"linux-arm64":  "cosmovisor-%s-linux-arm64.tar.gz",
		"darwin-amd64": "cosmovisor-%s-darwin-amd64.tar.gz",
	}

	key := fmt.Sprintf("%s-%s", osType, arch)
	template, exists := supportedCombinations[key]
	if !exists {
		panic(fmt.Sprintf("unsupported combination of OS and architecture: %s-%s", osType, arch))
	}

	url := fmt.Sprintf("https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%%2F%s/%s", version, fmt.Sprintf(template, version))

	// Check if the binary already exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Ensure the extracted path exists
		if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
			err := os.MkdirAll(extractedPath, os.ModePerm)
			if err != nil {
				panic(fmt.Sprintf("failed to create cosmovisor directory: %v", err))
			}
		}

		// Download and extract the tarball
		if err = io.DownloadAndExtractTarGz(url, tarballPath, extractedPath); err != nil {
			panic(fmt.Sprintf("failed to download and extract cosmovisor binary: %v", err))
		}

		// Set permissions for the binary
		err = os.Chmod(binaryPath, 0755)
		if err != nil {
			panic(fmt.Sprintf("failed to set permissions for cosmovisor binary: %v", err))
		}
	}

	return binaryPath
}
