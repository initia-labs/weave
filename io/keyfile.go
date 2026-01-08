package io

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/initia-labs/weave/crypto"
)

type Key struct {
	Address     string             `json:"address"`
	Mnemonic    string             `json:"mnemonic"`
	AddressType crypto.AddressType `json:"address_type"`
}

func NewKey(address, mnemonic string, addressType crypto.AddressType) *Key {
	return &Key{
		Address:     address,
		Mnemonic:    mnemonic,
		AddressType: addressType,
	}
}

func GenerateKey(hrp string, addressType crypto.AddressType) (*Key, error) {
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	address, err := crypto.MnemonicToBech32Address(hrp, mnemonic, addressType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address: %w", err)
	}

	return NewKey(address, mnemonic, addressType), nil
}

func RecoverKey(hrp, mnemonic string, addressType crypto.AddressType) (*Key, error) {
	address, err := crypto.MnemonicToBech32Address(hrp, mnemonic, addressType)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address: %w", err)
	}

	return &Key{
		Mnemonic: mnemonic,
		Address:  address,
	}, nil
}

type KeyFile map[string]*Key

func NewKeyFile() KeyFile {
	kf := make(KeyFile)
	return kf
}

func (k KeyFile) AddKey(name string, key *Key) {
	k[name] = key
}

func (k KeyFile) GetMnemonic(name string) string {
	return k[name].Mnemonic
}

func (k KeyFile) Write(filePath string) error {
	data, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling KeyFile to JSON: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// Load tries to load an existing key file into the struct if the file exists
func (k KeyFile) Load(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	err = json.Unmarshal(data, &k)
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return nil
}
