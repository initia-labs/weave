package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type KeyInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	PubKey   string `json:"pubkey"`
	Mnemonic string `json:"mnemonic"`
}

func MustUnmarshalKeyInfo(rawJson string) KeyInfo {
	var account KeyInfo
	err := json.Unmarshal([]byte(rawJson), &account)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal JSON: %v", err))
	}
	return account
}

// AddOrReplace adds or replaces a key using `initiad keys add <keyname> --keyring-backend test` with 'y' confirmation
func AddOrReplace(appName, keyname string) (string, error) {
	// Command to add the key: echo 'y' | initiad keys add <keyname> --keyring-backend test
	cmd := exec.Command(appName, "keys", "add", keyname, "--keyring-backend", "test", "--output", "json")

	// Simulate pressing 'y' for confirmation
	cmd.Stdin = bytes.NewBufferString("y\n")

	// Run the command and capture the output
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to add key for %s: %v, output: %s", keyname, err, string(outputBytes))
	}

	return string(outputBytes), nil
}

func MustAddOrReplaceKey(appName, keyname string) string {
	rawKey, err := AddOrReplace(appName, keyname)
	if err != nil {
		panic(err)
	}
	return rawKey
}

func DeleteKey(appName, keyname string) error {
	cmd := exec.Command(appName, "keys", "delete", keyname, "--keyring-backend", "test", "-y")
	return cmd.Run()
}

func MustDeleteKey(appName, keyname string) {
	if err := DeleteKey(appName, keyname); err != nil {
		panic(err)
	}
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
	cmd := exec.Command(appName, "keys", "add", keyname, "--recover", "--keyring-backend", "test", "--output", "json")

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

func MustRecoverKeyFromMnemonic(appName, keyname, mnemonic string) string {
	rawKey, err := RecoverKeyFromMnemonic(appName, keyname, mnemonic)
	if err != nil {
		panic(err)
	}
	return rawKey
}

func MustGenerateNewKeyInfo(appName, keyname string) KeyInfo {
	rawKey := MustAddOrReplaceKey(appName, keyname)
	MustDeleteKey(appName, keyname)
	return MustUnmarshalKeyInfo(rawKey)
}

func MustGetAddressFromMnemonic(appName, mnemonic string) string {
	keyname := "weave.DummyKey"
	rawKey := MustRecoverKeyFromMnemonic(appName, keyname, mnemonic)
	MustDeleteKey(appName, keyname)
	key := MustUnmarshalKeyInfo(rawKey)
	return key.Address
}
