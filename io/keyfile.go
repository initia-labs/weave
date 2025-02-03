package io

import (
	"encoding/json"
	"fmt"
	"os"
)

type KeyFile map[string]string

func NewKeyFile() *KeyFile {
	kf := make(KeyFile)
	return &kf
}

func (k *KeyFile) AddMnemonic(name, mnemonic string) {
	(*k)[name] = mnemonic
}

func (k *KeyFile) GetMnemonic(name string) string {
	return (*k)[name]
}

func (k *KeyFile) Write(filePath string) error {
	data, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling KeyFile to JSON: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// Load tries to load an existing key file into the struct if the file exists
func (k *KeyFile) Load(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	err = json.Unmarshal(data, k)
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return nil
}
