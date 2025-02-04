package io

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/initia-labs/weave/crypto"
)

type Wallet struct {
	Address  string `json:"address"`
	Mnemonic string `json:"mnemonic"`
}

func NewWallet(address, mnemonic string) *Wallet {
	return &Wallet{
		Address:  address,
		Mnemonic: mnemonic,
	}
}

func GenerateWallet(hrp string) (*Wallet, error) {
	mnemonic, err := crypto.GenerateMnemonic()
	if err != nil {
		return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	address, err := crypto.MnemonicToBech32Address(hrp, mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address: %w", err)
	}

	return &Wallet{
		Mnemonic: mnemonic,
		Address:  address,
	}, nil
}

func RecoverWalletFromMnemonic(hrp, mnemonic string) (*Wallet, error) {
	address, err := crypto.MnemonicToBech32Address(hrp, mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to derive address: %w", err)
	}

	return &Wallet{
		Mnemonic: mnemonic,
		Address:  address,
	}, nil
}

type KeyFile map[string]*Wallet

func NewKeyFile() *KeyFile {
	kf := make(KeyFile)
	return &kf
}

func (k *KeyFile) AddWallet(name string, wallet *Wallet) {
	(*k)[name] = wallet
}

func (k *KeyFile) GetMnemonic(name string) string {
	return (*k)[name].Mnemonic
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
