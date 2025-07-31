package cosmosutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type KeyInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	PubKey   string `json:"pubkey"`
	Mnemonic string `json:"mnemonic"`
}

func UnmarshalKeyInfo(rawJson string) (KeyInfo, error) {
	var account KeyInfo
	err := json.Unmarshal([]byte(rawJson), &account)
	if err != nil {
		return KeyInfo{}, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	return account, nil
}

// AddOrReplace adds or replaces a key using `initiad keys add <keyname> --keyring-backend test` with 'y' confirmation
func AddOrReplace(appName, keyname string) (string, error) {
	var cmd *exec.Cmd
	if strings.HasSuffix(appName, "celestia-appd") {
		cmd = exec.Command(appName, "keys", "add", keyname, "--keyring-backend", "test", "--output", "json")
	} else {
		cmd = exec.Command(appName, "keys", "add", keyname, "--coin-type", "118", "--key-type", "secp256k1", "--keyring-backend", "test", "--output", "json")
	}
	// Simulate pressing 'y' for confirmation
	cmd.Stdin = bytes.NewBufferString("y\n")

	// Run the command and capture the output
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to add key for %s: %v, output: %s", keyname, err, string(outputBytes))
	}

	return string(outputBytes), nil
}

func DeleteKey(appName, keyname string) error {
	cmd := exec.Command(appName, "keys", "delete", keyname, "--keyring-backend", "test", "-y")
	return cmd.Run()
}

// KeyExists checks if a key with the given keyName exists using `initiad keys show`
func KeyExists(appName, keyname string) bool {
	cmd := exec.Command(appName, "keys", "show", keyname, "--keyring-backend", "test")
	// Run the command and capture the output or error
	err := cmd.Run()
	return err == nil
}

// RecoverKeyFromMnemonic recovers or replaces a key using a mnemonic phrase
// If the key already exists, it will replace the key and confirm with 'y' before adding the mnemonic
func RecoverKeyFromMnemonic(appName, keyname, mnemonic string) (string, error) {
	// Check if the key already exists
	exists := KeyExists(appName, keyname)

	var inputBuffer bytes.Buffer
	if exists {
		// Simulate pressing 'y' for confirmation
		inputBuffer.WriteString("y\n")
	}

	// Add the mnemonic input after the confirmation (if any)
	inputBuffer.WriteString(mnemonic + "\n")

	var cmd *exec.Cmd
	if strings.HasSuffix(appName, "celestia-appd") {
		cmd = exec.Command(appName, "keys", "add", keyname, "--recover", "--keyring-backend", "test", "--output", "json")
	} else {
		cmd = exec.Command(appName, "keys", "add", keyname, "--coin-type", "118", "--key-type", "secp256k1", "--recover", "--keyring-backend", "test", "--output", "json")
	}

	// Pass the combined confirmation and mnemonic as input to the command
	cmd.Stdin = &inputBuffer

	// Run the command and capture the output
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to recover or replace key for %s: %v, output: %s", keyname, err, string(outputBytes))
	}

	// Return the command output if successful
	return string(outputBytes), nil
}

func GenerateNewKeyInfo(appName, keyname string) (KeyInfo, error) {
	rawKey, err := AddOrReplace(appName, keyname)
	if err != nil {
		return KeyInfo{}, err
	}
	if err = DeleteKey(appName, keyname); err != nil {
		return KeyInfo{}, err
	}
	return UnmarshalKeyInfo(rawKey)
}

func GetAddressFromMnemonic(appName, mnemonic string) (string, error) {
	keyname := "weave.DummyKey"
	rawKey, err := RecoverKeyFromMnemonic(appName, keyname, mnemonic)
	if err != nil {
		return "", err
	}
	if err := DeleteKey(appName, keyname); err != nil {
		return "", err
	}
	key, err := UnmarshalKeyInfo(rawKey)
	if err != nil {
		return "", err
	}
	return key.Address, nil
}

// OPInitRecoverKeyFromMnemonic recovers or replaces a key using a mnemonic phrase
// If the key already exists, it will replace the key and confirm with 'y' before adding the mnemonic
func OPInitRecoverKeyFromMnemonic(appName, keyname, mnemonic string, isCelestia bool, opInitHome string) (string, error) {
	// Check if the key already exists
	exists := OPInitKeyExist(appName, keyname, opInitHome)

	{
		var cmd *exec.Cmd
		var inputBuffer bytes.Buffer
		if exists {
			// Simulate pressing 'y' for confirmation
			inputBuffer.WriteString("y\n")
			cmd = exec.Command(appName, "keys", "delete", "weave-dummy", keyname, "--home", opInitHome)
			// Run the command and capture the output
			outputBytes, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("failed to delete key for %s: %v, output: %s", keyname, err, string(outputBytes))
			}

		}
	}
	var cmd *exec.Cmd
	var inputBuffer bytes.Buffer

	// Add the mnemonic input after the confirmation (if any)
	inputBuffer.WriteString(mnemonic + "\n")
	if isCelestia {
		cmd = exec.Command(appName, "keys", "add", "weave-dummy", keyname, "--coin-type", "118", "--key-type", "secp256k1", "--recover", "--bech32", "celestia", "--home", opInitHome)
	} else {
		cmd = exec.Command(appName, "keys", "add", "weave-dummy", keyname, "--coin-type", "118", "--key-type", "secp256k1", "--recover", "--home", opInitHome)
	}
	// Pass the combined confirmation and mnemonic as input to the command
	cmd.Stdin = &inputBuffer

	// Run the command and capture the output
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to recover or replace key for %s: %v, output: %s", keyname, err, string(outputBytes))
	}

	// Return the command output if successful
	return string(outputBytes), nil
}

func OPInitKeyExist(appName, keyname, opInitHome string) bool {
	cmd := exec.Command(appName, "keys", "show", "weave-dummy", keyname, "--home", opInitHome)
	// Run the command and capture the output or error
	err := cmd.Run()
	return err == nil
}

// OPInitGetAddressForKey retrieves the address for a given key using opinitd.
func OPInitGetAddressForKey(appName, keyname, opInitHome string) (string, error) {
	cmd := exec.Command(appName, "keys", "show", "weave-dummy", keyname, "--home", opInitHome)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get address for key %s: %v, output: %s", keyname, err, string(outputBytes))
	}

	// Parse the output to extract the address
	address, err := extractAddress(string(outputBytes), keyname)
	if err != nil {
		return "", fmt.Errorf("failed to extract address for %s: %v", keyname, err)
	}

	return address, nil
}

// OPInitGrantOracle grants oracle permissions to a specific address.
func OPInitGrantOracle(appName, address, opInitHome string) error {
	cmd := exec.Command(appName, "tx", "grant-oracle", address, "--home", opInitHome)
	if output, err := cmd.CombinedOutput(); err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "fee allowance already exists") {
			return nil
		}
		return fmt.Errorf("failed to grant oracle to address %s: %v (output: %s)", address, err, outputStr)
	}
	return nil
}

// extractAddress parses the command output to extract the key address.
func extractAddress(output, keyname string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, keyname+":") {
			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				return "", errors.New("invalid format for key address")
			}
			return strings.TrimSpace(parts[1]), nil
		}
	}
	return "", errors.New("key address not found in output")
}

// OPInitAddOrReplace adds or replaces a key using `opinitd keys add <keyname> --keyring-backend test`
// with 'y' confirmation
func OPInitAddOrReplace(appName, keyname string, isCelestia bool, opInitHome string) (string, error) {
	// Check if the key already exists
	exists := OPInitKeyExist(appName, keyname, opInitHome)
	{
		var cmd *exec.Cmd
		var inputBuffer bytes.Buffer
		if exists {
			// Simulate pressing 'y' for confirmation
			inputBuffer.WriteString("y\n")
			cmd = exec.Command(appName, "keys", "delete", "weave-dummy", keyname, "--home", opInitHome)
			// Run the command and capture the output
			outputBytes, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("failed to delete key for %s: %v, output: %s", keyname, err, string(outputBytes))
			}

		}
	}

	var cmd *exec.Cmd

	if isCelestia {
		cmd = exec.Command(appName, "keys", "add", "weave-dummy", keyname, "--coin-type", "118", "--key-type", "secp256k1", "--bech32", "celestia", "--home", opInitHome)
	} else {
		cmd = exec.Command(appName, "keys", "add", "weave-dummy", keyname, "--coin-type", "118", "--key-type", "secp256k1", "--home", opInitHome)
	}
	// Simulate pressing 'y' for confirmation
	cmd.Stdin = bytes.NewBufferString("y\n")

	// Run the command and capture the output
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to add key for %s: %v, output: %s", keyname, err, string(outputBytes))
	}

	return string(outputBytes), nil
}

func GetBinaryVersion(appName string) (string, error) {
	var cmd *exec.Cmd
	var inputBuffer bytes.Buffer
	// Simulate pressing 'y' for confirmation
	inputBuffer.WriteString("y\n")
	cmd = exec.Command(appName, "version")
	// Run the command and capture the output
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get binary version of %s: %v, output: %s", appName, err, string(outputBytes))
	}

	return strings.Trim(string(outputBytes), "\n"), nil
}

// SetSymlink sets a symbolic link in the parent directory pointing to the target binary.
func SetSymlink(targetPath string) error {
	// Resolve an absolute path for clarity
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of target: %v", err)
	}

	// Extract the base name of the target binary to create the symlink name automatically
	// Example: if the target is "~/.weave/data/opinitd@v0.1.0-test/opinitd", the symlink name will be "opinitd".
	binaryName := filepath.Base(absTargetPath)

	// Define the symlink path in the parent directory of the versioned directory
	symlinkPath := filepath.Join(filepath.Dir(filepath.Dir(absTargetPath)), binaryName)

	// Check if the symlink or file already exists
	if fileInfo, err := os.Lstat(symlinkPath); err == nil {
		// If the path exists and is a symlink
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			existingTarget, err := os.Readlink(symlinkPath)
			if err != nil {
				return fmt.Errorf("failed to read existing symlink: %v", err)
			}
			// If the symlink points to a different target, remove it
			if existingTarget != absTargetPath {
				if err := os.Remove(symlinkPath); err != nil {
					return fmt.Errorf("failed to remove existing symlink: %v", err)
				}
			} else {
				return nil
			}
		} else {
			// If the path is not a symlink (file or directory), remove it
			if err := os.Remove(symlinkPath); err != nil {
				return fmt.Errorf("failed to remove existing file or directory: %v", err)
			}
		}
	} else if !os.IsNotExist(err) {
		// If there's an error other than "not exist", return it
		return fmt.Errorf("failed to check existing file or directory: %v", err)
	}

	// Create the symlink
	if err := os.Symlink(absTargetPath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %v", err)
	}

	return nil
}
