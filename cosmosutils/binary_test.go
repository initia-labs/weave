package cosmosutils

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompareSemVer(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
	}{
		{
			v1:       "1.2.3",
			v2:       "1.2.2",
			expected: true,
		},
		{
			v1:       "2.0.0",
			v2:       "1.9.9",
			expected: true,
		},
		{
			v1:       "1.0.0",
			v2:       "1.0.0",
			expected: false,
		},
		{
			v1:       "1.2.0",
			v2:       "1.1.9",
			expected: true,
		},
		{
			v1:       "2.0.0",
			v2:       "1.9.9",
			expected: true,
		},
		{
			v1:       "1.0.0",
			v2:       "2.0.0",
			expected: false,
		},
		{
			name:     "complex prerelease identifiers",
			v1:       "1.0.0-alpha.2",
			v2:       "1.0.0-alpha.1",
			expected: true,
		},
		{
			name:     "complex prerelease identifiers reverse",
			v1:       "1.0.0-alpha.1",
			v2:       "1.0.0-alpha.2",
			expected: false,
		},
		{
			name:     "different prerelease identifiers",
			v1:       "1.0.0-beta.1",
			v2:       "1.0.0-alpha.2",
			expected: true,
		},
		{
			name:     "release vs complex prerelease",
			v1:       "1.0.0",
			v2:       "1.0.0-beta.1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareSemVer(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareSemVer(%s, %s) = %v, want %v",
					tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestGetLatestVersionFromReleases(t *testing.T) {
	// Get actual OS and arch
	currentOS, currentArch, err := getOSArch()
	if err != nil {
		t.Fatalf("Failed to get OS/arch: %v", err)
	}

	tests := []struct {
		name           string
		releases       []BinaryRelease
		expectedTag    string
		expectedURL    string
		expectedErrStr string
	}{
		{
			name:           "empty releases",
			releases:       []BinaryRelease{},
			expectedErrStr: "no releases found",
		},
		{
			name: "single compatible release",
			releases: []BinaryRelease{
				{
					TagName: "v1.0.0",
					Assets: []struct {
						BrowserDownloadURL string `json:"browser_download_url"`
					}{
						{BrowserDownloadURL: fmt.Sprintf("example.com/v1.0.0_%s_%s.tar.gz", currentOS, currentArch)},
					},
				},
			},
			expectedTag: "v1.0.0",
			expectedURL: fmt.Sprintf("example.com/v1.0.0_%s_%s.tar.gz", currentOS, currentArch),
		},
		{
			name: "multiple releases with different versions",
			releases: []BinaryRelease{
				{
					TagName: "v1.0.0",
					Assets: []struct {
						BrowserDownloadURL string `json:"browser_download_url"`
					}{
						{BrowserDownloadURL: fmt.Sprintf("example.com/v1.0.0_%s_%s.tar.gz", currentOS, currentArch)},
					},
				},
				{
					TagName: "v2.0.0",
					Assets: []struct {
						BrowserDownloadURL string `json:"browser_download_url"`
					}{
						{BrowserDownloadURL: fmt.Sprintf("example.com/v2.0.0_%s_%s.tar.gz", currentOS, currentArch)},
					},
				},
				{
					TagName: "v1.5.0",
					Assets: []struct {
						BrowserDownloadURL string `json:"browser_download_url"`
					}{
						{BrowserDownloadURL: fmt.Sprintf("example.com/v1.5.0_%s_%s.tar.gz", currentOS, currentArch)},
					},
				},
			},
			expectedTag: "v2.0.0",
			expectedURL: fmt.Sprintf("example.com/v2.0.0_%s_%s.tar.gz", currentOS, currentArch),
		},
		{
			name: "no compatible release for platform",
			releases: []BinaryRelease{
				{
					TagName: "v1.0.0",
					Assets: []struct {
						BrowserDownloadURL string `json:"browser_download_url"`
					}{
						{BrowserDownloadURL: "example.com/v1.0.0_Windows_x86_64.tar.gz"},
					},
				},
			},
			expectedErrStr: fmt.Sprintf("no compatible stable release found for %s_%s", currentOS, currentArch),
		},
		{
			name: "mixed compatible and incompatible releases",
			releases: []BinaryRelease{
				{
					TagName: "v1.0.0",
					Assets: []struct {
						BrowserDownloadURL string `json:"browser_download_url"`
					}{
						{BrowserDownloadURL: "example.com/v1.0.0_Windows_x86_64.tar.gz"},
					},
				},
				{
					TagName: "v2.0.0",
					Assets: []struct {
						BrowserDownloadURL string `json:"browser_download_url"`
					}{
						{BrowserDownloadURL: fmt.Sprintf("example.com/v2.0.0_%s_%s.tar.gz", currentOS, currentArch)},
					},
				},
			},
			expectedTag: "v2.0.0",
			expectedURL: fmt.Sprintf("example.com/v2.0.0_%s_%s.tar.gz", currentOS, currentArch),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, url, err := getLatestVersionFromReleases(tt.releases)

			if tt.expectedErrStr != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectedErrStr)
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErrStr) {
					t.Errorf("expected error containing %q, got %q", tt.expectedErrStr, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tag != tt.expectedTag {
				t.Errorf("expected tag %q, got %q", tt.expectedTag, tag)
			}

			if url != tt.expectedURL {
				t.Errorf("expected URL %q, got %q", tt.expectedURL, url)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "normal version",
			version:  "v1.2.3",
			expected: "v1.2.3",
		},
		{
			name:     "version with prerelease",
			version:  "v1.2.3-beta",
			expected: "v1.2.3-beta",
		},
		{
			name:     "version with prerelease and additional text",
			version:  "v1.2.3-beta-123",
			expected: "v1.2.3-beta-123",
		},
		{
			name:     "version with refs/tags prefix",
			version:  "refs/tags/v1.2.3",
			expected: "v1.2.3",
		},
		{
			name:     "version with refs/tags prefix and additional text",
			version:  "refs/tags/v1.2.355",
			expected: "v1.2.355",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeVersion(tt.version)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%s) = %s, want %s", tt.version, result, tt.expected)
			}
		})
	}
}
