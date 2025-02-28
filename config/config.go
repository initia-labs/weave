package config

import (
	"encoding/json"
	"fmt"
	"github.com/initia-labs/weave/io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/viper"

	"github.com/initia-labs/weave/common"
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

func GetCommandImageURL(commandName string) string {
	return GetConfig(fmt.Sprintf("%s.image_url", commandName)).(string)
}

func GetCommandHome(commandName string) string {
	return GetConfig(fmt.Sprintf("%s.home", commandName)).(string)
}

func SetCommandImageURL(commandName string, imageURL string) error {
	return SetConfig(fmt.Sprintf("%s.image_url", commandName), imageURL)
}

func SetCommandHome(commandName string, home string) error {
	return SetConfig(fmt.Sprintf("%s.home", commandName), home)
}

const DefaultConfigTemplate = `{}`

type GasStationKey struct {
	InitiaAddress   string `json:"initia_address"`
	CelestiaAddress string `json:"celestia_address"`
	Mnemonic        string `json:"mnemonic"`
}

func RecoverGasStationKey(mnemonic string) (*GasStationKey, error) {
	initiaKey, err := io.RecoverKey("init", mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to recover initia gas station key: %v", err)
	}

	celestiaKey, err := io.RecoverKey("celestia", mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to recover celestia gas station key: %v", err)
	}

	return &GasStationKey{
		InitiaAddress:   initiaKey.Address,
		CelestiaAddress: celestiaKey.Address,
		Mnemonic:        mnemonic,
	}, nil
}
