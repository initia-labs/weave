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

	data := GetConfig("common.gas_station")
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json: %v", err)
	}

	var gasKey GasStationKey
	err = json.Unmarshal(jsonData, &gasKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %v", err)
	}

	if gasKey.Mnemonic != "" {
		updated := false

		// If coin type is not set, try to detect it by testing both 60 and 118
		if gasKey.CoinType == nil {
			// Try coin type 60 first (EVM)
			addr60, err60 := crypto.MnemonicToBech32AddressWithCoinType("init", gasKey.Mnemonic, 60)
			// Try coin type 118 (Cosmos)
			addr118, err118 := crypto.MnemonicToBech32AddressWithCoinType("init", gasKey.Mnemonic, 118)

			// If we have a stored initia address, match against it
			if gasKey.InitiaAddress != "" {
				if err60 == nil && addr60 == gasKey.InitiaAddress {
					coinType := 60
					gasKey.CoinType = &coinType
					updated = true
				} else if err118 == nil && addr118 == gasKey.InitiaAddress {
					coinType := 118
					gasKey.CoinType = &coinType
					updated = true
				}
				// If neither matches the stored address, leave CoinType nil to prevent overwriting
			} else {
				// If no stored address, default to 118 for existing configs
				coinType := 118
				gasKey.CoinType = &coinType
				updated = true
			}
		}

		// Only proceed with address recovery if coin type was successfully determined
		if gasKey.CoinType != nil {
			// Recover initia address using the determined coin type
			initiaAddress, err := crypto.MnemonicToBech32AddressWithCoinType("init", gasKey.Mnemonic, *gasKey.CoinType)
			if err != nil {
				return nil, fmt.Errorf("failed to recover initia gas station key: %v", err)
			}

			celestiaKey, err := io.RecoverKey("celestia", gasKey.Mnemonic, crypto.CosmosAddressType)
			if err != nil {
				return nil, fmt.Errorf("failed to recover celestia gas station key: %v", err)
			}

			// Only update InitiaAddress if it matches the newly derived address or if empty
			if gasKey.InitiaAddress == "" || gasKey.InitiaAddress == initiaAddress {
				if gasKey.InitiaAddress != initiaAddress {
					gasKey.InitiaAddress = initiaAddress
					updated = true
				}
			}

			if gasKey.CelestiaAddress != celestiaKey.Address {
				gasKey.CelestiaAddress = celestiaKey.Address
				updated = true
			}

			if updated {
				err := SetConfig("common.gas_station", gasKey)
				if err != nil {
					return nil, fmt.Errorf("failed to persist gas station addresses: %v", err)
				}
			}
		}
	}

	// Ensure coin type is always set and valid
	if gasKey.CoinType == nil || *gasKey.CoinType == 0 {
		return nil, fmt.Errorf("gas station coin type not properly configured")
	}

	return &gasKey, nil
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
