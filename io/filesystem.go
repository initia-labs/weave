package io

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/initia-labs/weave/client"
)

// FileOrFolderExists checks if a file or folder exists at the given path.
func FileOrFolderExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func DownloadAndExtractTarGz(url, tarballPath, extractedPath string) error {
	httpClient := client.NewHTTPClient()
	if err := httpClient.DownloadFile(url, tarballPath, nil, nil); err != nil {
		return err
	}

	if err := ExtractTarGz(tarballPath, extractedPath); err != nil {
		return err
	}

	if err := os.Remove(tarballPath); err != nil {
		return fmt.Errorf("failed to remove tarball file: %v", err)
	}

	return nil
}

func ExtractTarGz(src string, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			file, err := os.Create(target)
			if err != nil {
				return err
			}
			_, err = io.Copy(file, tarReader)
			if err != nil {
				return err
			}
			err = file.Close()
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown type: %c", header.Typeflag)
		}
	}
	return nil
}

func SetLibraryPaths(binaryDir string) error {
	envKey, envValue, err := LibraryPathEnv(binaryDir)
	if err != nil {
		return err
	}
	if err := os.Setenv(envKey, envValue); err != nil {
		return fmt.Errorf("failed to set %s: %v", envKey, err)
	}
	return nil
}

func LibraryPathEnv(binaryDir string) (string, string, error) {
	envKey, err := libraryPathKey()
	if err != nil {
		return "", "", err
	}
	return envKey, buildLibraryPath(binaryDir, os.Getenv(envKey)), nil
}

func WithLibraryPathEnv(env []string, binaryDir string) ([]string, error) {
	envKey, err := libraryPathKey()
	if err != nil {
		return nil, err
	}
	envValue := buildLibraryPath(binaryDir, getEnvValue(env, envKey))
	return upsertEnv(env, envKey, envValue), nil
}

func libraryPathKey() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "DYLD_LIBRARY_PATH", nil
	case "linux":
		return "LD_LIBRARY_PATH", nil
	default:
		return "", fmt.Errorf("unsupported OS for setting library paths: %v", runtime.GOOS)
	}
}

func buildLibraryPath(binaryDir, existing string) string {
	paths := []string{binaryDir}
	if existing == "" {
		return strings.Join(paths, string(os.PathListSeparator))
	}
	existingPaths := strings.Split(existing, string(os.PathListSeparator))
	for _, p := range existingPaths {
		if p == "" || p == binaryDir {
			continue
		}
		paths = append(paths, p)
	}
	return strings.Join(paths, string(os.PathListSeparator))
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, item := range env {
		if strings.HasPrefix(item, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func getEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func WriteFile(path, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create or open file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write content to file: %v", err)
	}

	return nil
}

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	return nil
}

func DeleteDirectory(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("failed to delete directory: %v", err)
	}
	return nil
}

// CopyDirectory uses the cp -r command to copy files or directories from src to des.
func CopyDirectory(src, des string) error {
	// Check if destination exists
	if _, err := os.Stat(des); err == nil {
		// Remove the contents of the destination directory
		err := os.RemoveAll(des)
		if err != nil {
			return fmt.Errorf("could not clear destination directory: %v", err)
		}
	}

	// Now, perform the copy
	cmd := exec.Command("cp", "-r", src, des)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not run cp command: %v, output: %s", err, string(output))
	}
	return nil
}
