package utils

import (
	"bytes"
	"fmt"
	"os/exec"
)

// AddOrReplace adds or replaces a key using `initiad keys add <keyname> --keyring-backend test` with 'y' confirmation
func AddOrReplace(appName, keyname string) (string, error) {
	// Command to add the key: echo 'y' | initiad keys add <keyname> --keyring-backend test
	cmd := exec.Command(appName, "keys", "add", keyname, "--keyring-backend", "test")

	// Simulate pressing 'y' for confirmation
	cmd.Stdin = bytes.NewBufferString("y\n")

	// Run the command and capture the output
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to add key for %s: %v, output: %s", keyname, err, string(outputBytes))
	}

	return string(outputBytes), nil
}

// KeyExists checks if a key with the given keyName exists using `initiad keys show`
func KeyExists(appName, keyname string) bool {
	// Command to show the key: initiad keys show <keyname> --keyring-backend test
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
		// If the key exists, print a message about replacing it and add 'y' confirmation
		fmt.Printf("Key %s already exists, replacing it...\n", keyname)
		// Simulate pressing 'y' for confirmation
		inputBuffer.WriteString("y\n")
	}

	// Add the mnemonic input after the confirmation (if any)
	inputBuffer.WriteString(mnemonic + "\n")

	// Command to recover (or replace) the key: initiad keys add <keyname> --recover --keyring-backend test
	cmd := exec.Command(appName, "keys", "add", keyname, "--recover", "--keyring-backend", "test")

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
