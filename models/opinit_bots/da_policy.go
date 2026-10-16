package opinit_bots

import (
	"fmt"

	"github.com/btcsuite/btcutil/bech32"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/types"
)

// ValidateExecutorDA validates the DA used for a new executor initialization.
func ValidateExecutorDA(config ExecutorConfig) error {
	return common.ValidateNewDAPrefix(config.DANode.Bech32Prefix)
}

func pendingBatchPrefix(info BotInfo) (string, error) {
	switch info.DALayer {
	case string(InitiaLayerOption):
		return "init", nil
	case string(CelestiaLayerOption):
		return "celestia", nil
	default:
		return "", fmt.Errorf("unsupported Batch Submitter DA layer %q: expected Initia or Celestia", info.DALayer)
	}
}

func validatePendingBotKeys(infos []BotInfo) error {
	for _, info := range infos {
		if info.BotName != BatchSubmitter || !info.IsNewKey() {
			continue
		}
		prefix, err := pendingBatchPrefix(info)
		if err != nil {
			return err
		}
		if err := common.ValidateNewDAPrefix(prefix); err != nil {
			return err
		}
	}
	return nil
}

func validateExecutorStateDA(state OPInitBotsState) error {
	if !state.InitExecutorBot {
		return nil
	}
	prefix := state.botConfig["da_node.bech32_prefix"]
	if err := common.ValidateNewDAPrefix(prefix); err != nil {
		return err
	}
	for _, info := range state.BotInfos {
		if info.BotName != BatchSubmitter || !info.IsNewKey() {
			continue
		}
		keyPrefix, err := pendingBatchPrefix(info)
		if err != nil {
			return err
		}
		if keyPrefix != prefix {
			return fmt.Errorf("Batch Submitter DA layer %q does not match executor DA prefix %q", info.DALayer, prefix)
		}
	}
	return nil
}

func applyExistingExecutorConfig(state *OPInitBotsState, config ExecutorConfig) {
	if state.botConfig == nil {
		state.botConfig = make(map[string]string)
	}
	state.botConfig["l1_node.chain_id"] = config.L1Node.ChainID
	state.botConfig["l1_node.rpc_address"] = config.L1Node.RPCAddress
	state.botConfig["l1_node.gas_price"] = config.L1Node.GasPrice
	state.botConfig["l2_node.chain_id"] = config.L2Node.ChainID
	state.botConfig["l2_node.rpc_address"] = config.L2Node.RPCAddress
	state.botConfig["da_node.chain_id"] = config.DANode.ChainID
	state.botConfig["da_node.rpc_address"] = config.DANode.RPCAddress
	state.botConfig["da_node.bech32_prefix"] = config.DANode.Bech32Prefix
	state.botConfig["da_node.gas_price"] = config.DANode.GasPrice
	state.daIsCelestia = config.DANode.Bech32Prefix == "celestia"
}

// resolveArtifactDALayer identifies artifact DA without imposing the current build's policy.
// Explicit targets take precedence over legacy L1 addresses.
func resolveArtifactDALayer(config *types.MinitiaConfig) (DALayerOption, error) {
	if config == nil {
		return "", fmt.Errorf("missing minitia config")
	}
	target := ""
	if config.OpBridge != nil {
		target = config.OpBridge.BatchSubmissionTarget
	}
	var layer DALayerOption
	switch target {
	case "INITIA":
		layer = InitiaLayerOption
	case "CELESTIA":
		layer = CelestiaLayerOption
	case "":
	default:
		return "", fmt.Errorf("unsupported DA target %q: expected INITIA or CELESTIA", target)
	}
	address := ""
	if config.SystemKeys != nil && config.SystemKeys.BatchSubmitter != nil {
		address = config.SystemKeys.BatchSubmitter.DAAddress
		if address == "" && target == "" {
			address = config.SystemKeys.BatchSubmitter.L1Address
		}
	}
	if address == "" {
		if layer != "" {
			return layer, nil
		}
		return "", fmt.Errorf("missing Batch Submitter DA address for legacy artifact")
	}
	prefix, data, err := bech32.Decode(address)
	if err != nil || len(data) == 0 {
		return "", fmt.Errorf("invalid Batch Submitter DA address %q", address)
	}
	var addressLayer DALayerOption
	switch prefix {
	case "init":
		addressLayer = InitiaLayerOption
	case "celestia":
		addressLayer = CelestiaLayerOption
	default:
		return "", fmt.Errorf("unsupported Batch Submitter DA address prefix %q", prefix)
	}
	if layer != "" && layer != addressLayer {
		return "", fmt.Errorf("DA target %q conflicts with Batch Submitter DA address prefix %q", target, prefix)
	}
	return addressLayer, nil
}

func validateArtifactKeys(config *types.MinitiaConfig) (DALayerOption, error) {
	if config == nil || config.SystemKeys == nil {
		return "", fmt.Errorf("missing artifact system keys")
	}
	for _, entry := range []struct {
		name    BotName
		account *types.SystemAccount
	}{
		{BridgeExecutor, config.SystemKeys.BridgeExecutor},
		{OutputSubmitter, config.SystemKeys.OutputSubmitter},
		{BatchSubmitter, config.SystemKeys.BatchSubmitter},
		{Challenger, config.SystemKeys.Challenger},
	} {
		if entry.account == nil {
			return "", fmt.Errorf("missing artifact account for %s", entry.name)
		}
		if err := common.ValidateMnemonic(entry.account.Mnemonic); err != nil {
			return "", fmt.Errorf("invalid artifact mnemonic for %s: %w", entry.name, err)
		}
	}
	layer, err := resolveArtifactDALayer(config)
	if err != nil {
		return "", err
	}
	prefix, err := pendingBatchPrefix(BotInfo{DALayer: string(layer)})
	if err != nil {
		return "", err
	}
	if err := common.ValidateNewDAPrefix(prefix); err != nil {
		return "", err
	}
	return layer, nil
}
