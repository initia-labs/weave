package relayer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/registry"
)

func UpdateClientFromConfig() error {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(userHome, common.HermesHome, "config.toml")
	weaveDataPath := filepath.Join(userHome, common.WeaveDataDirectory)
	hermesBinaryPath := filepath.Join(weaveDataPath, "hermes")

	tomlData, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config Config
	err = toml.Unmarshal(tomlData, &config)
	if err != nil {
		return err
	}
	if len(config.Chains) < 2 {
		return fmt.Errorf("invalid configuration: missing chain configuration")
	}

	var chainRegistry *registry.ChainRegistry

	// Avoid panic until mainnet launches
	testnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Testnet)
	if err != nil {
		return fmt.Errorf("error loading testnet registry: %v", err)
	}
	if config.Chains[0].ID == testnetRegistry.GetChainId() {
		chainRegistry = testnetRegistry
	} else {
		mainnetRegistry, err := registry.GetChainRegistry(registry.InitiaL1Mainnet)
		if err != nil {
			return fmt.Errorf("error loading mainnet registry: %v", err)
		}
		if config.Chains[0].ID == mainnetRegistry.GetChainId() {
			chainRegistry = mainnetRegistry
		}
	}

	if chainRegistry == nil {
		return fmt.Errorf("chain registry not found")
	}

	clientIds := make(map[string]bool)
	for _, channel := range config.Chains[0].PacketFilter.List {
		connection, err := chainRegistry.GetCounterpartyClientId(channel[0], channel[1])
		if err != nil {
			return err
		}
		clientIds[connection.Connection.Counterparty.ClientID] = true
	}
	te := cosmosutils.NewHermesTxExecutor(hermesBinaryPath)

	for clientId := range clientIds {
		fmt.Printf("Updating IBC client: %s of network: %s\n", clientId, config.Chains[1].ID)
		_, err := te.UpdateClient(clientId, config.Chains[1].ID)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully updated IBC client: %s of network: %s\n", clientId, config.Chains[1].ID)
	}
	return nil
}
