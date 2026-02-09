package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/viper"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/crypto"
	"github.com/initia-labs/weave/io"
)

var DevMode string

func InitializeConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	configPath := filepath.Join(homeDir, common.WeaveConfigFile)
	if err := os.MkdirAll(filepath.Dir(configPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	dataPath := filepath.Join(homeDir, common.WeaveDataDirectory)
	if err := os.MkdirAll(dataPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	logPath := filepath.Join(homeDir, common.WeaveLogDirectory)
	if err := os.MkdirAll(logPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfigFile(configPath); err != nil {
			return fmt.Errorf("failed to create default config file: %v", err)
		}
	}

	return LoadConfig()
}

func createDefaultConfigFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(DefaultConfigTemplate)
	if err != nil {
		return fmt.Errorf("failed to write to config file: %v", err)
	}

	return nil
}

func LoadConfig() error {
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	return nil
}

func GetConfig(key string) interface{} {
	return viper.Get(key)
}

func SetConfig(key string, value interface{}) error {
	viper.Set(key, value)
	return WriteConfig()
}

func WriteConfig() error {
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	return nil
}

func IsFirstTimeSetup() bool {
	return viper.Get("common.gas_station") == nil
}

func GetGasStationKey() (*GasStationKey, error) {
	if IsFirstTimeSetup() {
		return nil, fmt.Errorf("gas station key not exists")
	}

	gasKey, err := loadGasStationKeyFromConfig()
	if err != nil {
		return nil, err
	}

	if gasKey.Mnemonic != "" {
		if err := ensureGasStationKeyIntegrity(gasKey); err != nil {
			return nil, err
		}
	}

	if err := validateGasStationKey(gasKey); err != nil {
		return nil, err
	}

	return gasKey, nil
}

func loadGasStationKeyFromConfig() (*GasStationKey, error) {
	data := GetConfig("common.gas_station")
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json: %v", err)
	}

	var gasKey GasStationKey
	if err := json.Unmarshal(jsonData, &gasKey); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %v", err)
	}

	return &gasKey, nil
}

func ensureGasStationKeyIntegrity(gasKey *GasStationKey) error {
	updated := false

	// Detect and set coin type if missing
	if gasKey.CoinType == nil {
		coinTypeSet, err := detectAndSetCoinType(gasKey)
		if err != nil {
			return err
		}
		updated = coinTypeSet
	}

	// Only recover addresses if coin type is set
	if gasKey.CoinType != nil {
		addressesUpdated, err := recoverAndUpdateAddresses(gasKey)
		if err != nil {
			return err
		}
		updated = updated || addressesUpdated
	}

	if updated {
		if err := SetConfig("common.gas_station", gasKey); err != nil {
			return fmt.Errorf("failed to persist gas station addresses: %v", err)
		}
	}

	return nil
}

func detectAndSetCoinType(gasKey *GasStationKey) (bool, error) {
	// If we have a stored address, try to match it
	if gasKey.InitiaAddress != "" {
		if coinType, ok := matchCoinTypeToAddress(gasKey.Mnemonic, gasKey.InitiaAddress); ok {
			gasKey.CoinType = &coinType
			return true, nil
		}
		// If neither matches, leave CoinType nil to prevent overwriting
		return false, nil
	}

	// Default to 118 for existing configs without stored address
	coinType := 118
	gasKey.CoinType = &coinType
	return true, nil
}

func matchCoinTypeToAddress(mnemonic, storedAddress string) (int, bool) {
	// Try coin type 60 (EVM)
	if addr60, err := crypto.MnemonicToBech32AddressWithCoinType("init", mnemonic, 60); err == nil && addr60 == storedAddress {
		return 60, true
	}

	// Try coin type 118 (Cosmos)
	if addr118, err := crypto.MnemonicToBech32AddressWithCoinType("init", mnemonic, 118); err == nil && addr118 == storedAddress {
		return 118, true
	}

	return 0, false
}

func recoverAndUpdateAddresses(gasKey *GasStationKey) (bool, error) {
	updated := false

	// Recover initia address
	initiaAddress, err := crypto.MnemonicToBech32AddressWithCoinType("init", gasKey.Mnemonic, *gasKey.CoinType)
	if err != nil {
		return false, fmt.Errorf("failed to recover initia gas station key: %v", err)
	}

	// Recover celestia address
	celestiaKey, err := io.RecoverKey("celestia", gasKey.Mnemonic, crypto.CosmosAddressType)
	if err != nil {
		return false, fmt.Errorf("failed to recover celestia gas station key: %v", err)
	}

	// Update initia address if empty or matches newly derived address
	if gasKey.InitiaAddress == "" || gasKey.InitiaAddress == initiaAddress {
		if gasKey.InitiaAddress != initiaAddress {
			gasKey.InitiaAddress = initiaAddress
			updated = true
		}
	}

	// Update celestia address if changed
	if gasKey.CelestiaAddress != celestiaKey.Address {
		gasKey.CelestiaAddress = celestiaKey.Address
		updated = true
	}

	return updated, nil
}

func validateGasStationKey(gasKey *GasStationKey) error {
	if gasKey.CoinType == nil || *gasKey.CoinType == 0 {
		return fmt.Errorf("gas station coin type not properly configured")
	}
	return nil
}

func AnalyticsOptOut() bool {
	// In dev mode, always opt out
	if DevMode == "true" {
		return true
	}

	if GetConfig("common.analytics_opt_out") == nil {
		_ = SetConfig("common.analytics_opt_out", false)
		return false
	}

	return GetConfig("common.analytics_opt_out").(bool)
}

func GetAnalyticsDeviceID() string {
	if GetConfig("common.analytics_device_id") == nil {
		deviceID := uuid.New().String()
		_ = SetConfig("common.analytics_device_id", deviceID)
		return deviceID
	}

	return GetConfig("common.analytics_device_id").(string)
}

func SetAnalyticsOptOut(optOut bool) error {
	return SetConfig("common.analytics_opt_out", optOut)
}

const DefaultConfigTemplate = `{}`

type GasStationKey struct {
	InitiaAddress   string `json:"initia_address"`
	CelestiaAddress string `json:"celestia_address"`
	Mnemonic        string `json:"mnemonic"`
	CoinType        *int   `json:"coin_type,omitempty"`
}

func RecoverGasStationKey(mnemonic string) (*GasStationKey, error) {
	initiaKey, err := io.RecoverKey("init", mnemonic, crypto.EVMAddressType)
	if err != nil {
		return nil, fmt.Errorf("failed to recover initia gas station key: %v", err)
	}

	celestiaKey, err := io.RecoverKey("celestia", mnemonic, crypto.CosmosAddressType)
	if err != nil {
		return nil, fmt.Errorf("failed to recover celestia gas station key: %v", err)
	}

	// Default coin type for new configs is 60
	coinType := 60
	return &GasStationKey{
		InitiaAddress:   initiaKey.Address,
		CelestiaAddress: celestiaKey.Address,
		Mnemonic:        mnemonic,
		CoinType:        &coinType,
	}, nil
}
