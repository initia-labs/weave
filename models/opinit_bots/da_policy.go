package opinit_bots

import (
	"fmt"

	"github.com/initia-labs/weave/common"
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
