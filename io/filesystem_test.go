package io

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock HTTP client for testing
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) DownloadFile(url, dest string, progress, totalSize *int64) error {
	args := m.Called(url, dest, progress, totalSize)
	return args.Error(0)
}

func TestFileOrFolderExists(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		exists bool
	}{
		{"File exists", "./testfile", true},
		{"File does not exist", "./nonexistent", false},
	}

	// Create a test file for this example
	t.Run("FileExists", func(t *testing.T) {
		f, err := os.Create("./testfile")
		assert.NoError(t, err)
		f.Close()
		defer os.Remove("./testfile")

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.exists, FileOrFolderExists(tt.path))
			})
		}
	})
}

func TestDownloadAndExtractTarGz(t *testing.T) {
	client := new(MockHTTPClient)

	t.Run("TestDownloadAndExtractTarGzFailure", func(t *testing.T) {
		client.On("DownloadFile", "http://example.com/tarball.tar.gz", "./test.tar.gz", nil, nil).Return(assert.AnError)
		err := DownloadAndExtractTarGz("http://example.com/tarball.tar.gz", "./test.tar.gz", "./testdir")
		assert.Error(t, err)
	})
}

func TestExtractTarGz(t *testing.T) {
	t.Run("TestExtractTarGzFailure", func(t *testing.T) {
		// Test invalid tarball
		err := ExtractTarGz("./invalid.tar.gz", "./invalid")
		assert.Error(t, err)
	})

	t.Run("PreservesExtractedFileMode", func(t *testing.T) {
		tmpDir := t.TempDir()
		tarballPath := filepath.Join(tmpDir, "test.tar.gz")
		extractDir := filepath.Join(tmpDir, "extract")

		file, err := os.Create(tarballPath)
		assert.NoError(t, err)

		gzw := gzip.NewWriter(file)
		tw := tar.NewWriter(gzw)

		content := []byte("#!/bin/sh\necho ok\n")
		header := &tar.Header{
			Name: "minitiad",
			Mode: 0o755,
			Size: int64(len(content)),
		}
		assert.NoError(t, tw.WriteHeader(header))
		_, err = tw.Write(content)
		assert.NoError(t, err)
		assert.NoError(t, tw.Close())
		assert.NoError(t, gzw.Close())
		assert.NoError(t, file.Close())

		err = ExtractTarGz(tarballPath, extractDir)
		assert.NoError(t, err)

		info, err := os.Stat(filepath.Join(extractDir, "minitiad"))
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("RejectsPathTraversalEntries", func(t *testing.T) {
		tmpDir := t.TempDir()
		tarballPath := filepath.Join(tmpDir, "test.tar.gz")
		extractDir := filepath.Join(tmpDir, "extract")

		file, err := os.Create(tarballPath)
		assert.NoError(t, err)

		gzw := gzip.NewWriter(file)
		tw := tar.NewWriter(gzw)

		content := []byte("bad\n")
		header := &tar.Header{
			Name: "../escape",
			Mode: 0o644,
			Size: int64(len(content)),
		}
		assert.NoError(t, tw.WriteHeader(header))
		_, err = tw.Write(content)
		assert.NoError(t, err)
		assert.NoError(t, tw.Close())
		assert.NoError(t, gzw.Close())
		assert.NoError(t, file.Close())

		err = ExtractTarGz(tarballPath, extractDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsafe archive entry path")
	})
}

func TestSetLibraryPaths(t *testing.T) {
	t.Run("TestSetLibraryPathsLinux", func(t *testing.T) {
		// Mock Linux environment variable setting
		if err := os.Setenv("GOOS", "linux"); err != nil {
			t.Fatal("Failed to set GOOS environment variable")
		}
		// Normally, you'd check the environment variable being set
		SetLibraryPaths("./somepath")
	})

	t.Run("TestSetLibraryPathsDarwin", func(t *testing.T) {
		// Mock Darwin environment variable setting
		if err := os.Setenv("GOOS", "darwin"); err != nil {
			t.Fatal("Failed to set GOOS environment variable")
		}
		SetLibraryPaths("./somepath")
	})
}

func TestWriteFile(t *testing.T) {
	t.Run("TestWriteFileSuccess", func(t *testing.T) {
		err := WriteFile("./testfile.txt", "Hello, World!")
		assert.NoError(t, err)
		defer os.Remove("./testfile.txt")

		// Check file content
		content, err := os.ReadFile("./testfile.txt")
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World!", string(content))
	})

	t.Run("TestWriteFileFailure", func(t *testing.T) {
		err := WriteFile("/invalid/path/to/file.txt", "Hello, World!")
		assert.Error(t, err)
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("TestDeleteFileSuccess", func(t *testing.T) {
		_, err := os.Create("./fileToDelete.txt")
		assert.NoError(t, err)
		defer os.Remove("./fileToDelete.txt")

		err = DeleteFile("./fileToDelete.txt")
		assert.NoError(t, err)
		_, err = os.Stat("./fileToDelete.txt")
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("TestDeleteFileFailure", func(t *testing.T) {
		err := DeleteFile("./nonexistent.txt")
		assert.Error(t, err)
	})
}

func TestDeleteDirectory(t *testing.T) {
	t.Run("TestDeleteDirectorySuccess", func(t *testing.T) {
		err := os.Mkdir("./testdir", os.ModePerm)
		assert.NoError(t, err)
		defer os.RemoveAll("./testdir")

		err = DeleteDirectory("./testdir")
		assert.NoError(t, err)
		_, err = os.Stat("./testdir")
		assert.True(t, os.IsNotExist(err))
	})
}

func TestCopyDirectory(t *testing.T) {
	t.Run("TestCopyDirectorySuccess", func(t *testing.T) {
		err := os.Mkdir("./src", os.ModePerm)
		assert.NoError(t, err)
		defer os.RemoveAll("./src")

		err = os.Mkdir("./des", os.ModePerm)
		assert.NoError(t, err)
		defer os.RemoveAll("./des")

		err = CopyDirectory("./src", "./des")
		assert.NoError(t, err)
	})

	t.Run("TestCopyDirectoryFailure", func(t *testing.T) {
		err := CopyDirectory("./nonexistentdir", "./des")
		assert.Error(t, err)
	})
}
