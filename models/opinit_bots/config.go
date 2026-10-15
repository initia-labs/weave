package opinit_bots

import (
	"encoding/json"
	"fmt"

	"github.com/initia-labs/weave/crypto"
	weaveio "github.com/initia-labs/weave/io"
)

type NodeConfig struct {
	ChainID      string `json:"chain_id"`
	Bech32Prefix string `json:"bech32_prefix"`
	RPCAddress   string `json:"rpc_address"`
}

type ChallengerConfig struct {
	Version                int          `json:"version"`
	Server                 ServerConfig `json:"server"`
	L1Node                 NodeConfig   `json:"l1_node"`
	L2Node                 NodeConfig   `json:"l2_node"`
	L1StartHeight          int          `json:"l1_start_height"`
	L2StartHeight          int          `json:"l2_start_height"`
	DisableAutoSetL1Height bool         `json:"disable_auto_set_l1_height"`
}

type NodeSettings struct {
	ChainID       string  `json:"chain_id"`
	Bech32Prefix  string  `json:"bech32_prefix"`
	RPCAddress    string  `json:"rpc_address"`
	GasPrice      string  `json:"gas_price"`
	GasAdjustment float64 `json:"gas_adjustment"`
	TxTimeout     int     `json:"tx_timeout"`
}

type ServerConfig struct {
	Address      string `json:"address"`
	AllowOrigins string `json:"allow_origins"`
	AllowHeaders string `json:"allow_headers"`
	AllowMethods string `json:"allow_methods"`
}

type ExecutorConfig struct {
	Version                       int          `json:"version"`
	Server                        ServerConfig `json:"server"`
	L1Node                        NodeSettings `json:"l1_node"`
	L2Node                        NodeSettings `json:"l2_node"`
	DANode                        NodeSettings `json:"da_node"`
	BridgeExecutor                string       `json:"bridge_executor"`
	OracleBridgeExecutor          string       `json:"oracle_bridge_executor"`
	DisableOutputSubmitter        bool         `json:"disable_output_submitter"`
	DisableBatchSubmitter         bool         `json:"disable_batch_submitter"`
	MaxChunks                     int          `json:"max_chunks"`
	MaxChunkSize                  int          `json:"max_chunk_size"`
	MaxSubmissionTime             int          `json:"max_submission_time"`
	DisableAutoSetL1Height        bool         `json:"disable_auto_set_l1_height"`
	L1StartHeight                 int          `json:"l1_start_height"`
	L2StartHeight                 int          `json:"l2_start_height"`
	BatchStartHeight              int          `json:"batch_start_height"`
	DisableDeleteFutureWithdrawal bool         `json:"disable_delete_future_withdrawal"`
}

func GenerateMnemonicKeyfile(rawConfig []byte, botName string) (weaveio.KeyFile, error) {
	keyFile := weaveio.NewKeyFile()

	switch botName {
	case "executor":
		var config ExecutorConfig
		err := json.Unmarshal(rawConfig, &config)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal executor config: %v", err)
		}
		if err := ValidateExecutorDA(config); err != nil {
			return nil, err
		}

		bridgeExecutor, err := weaveio.GenerateKey("init", crypto.CosmosAddressType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate bridge executor mnemonic: %w", err)
		}
		keyFile.AddKey(BridgeExecutorKeyName, bridgeExecutor)

		outputSubmitter, err := weaveio.GenerateKey("init", crypto.CosmosAddressType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate output submitter mnemonic: %w", err)
		}
		keyFile.AddKey(OutputSubmitterKeyName, outputSubmitter)

		batchSubmitter, err := weaveio.GenerateKey(config.DANode.Bech32Prefix, crypto.CosmosAddressType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate batch submitter mnemonic: %w", err)
		}
		keyFile.AddKey(BatchSubmitterKeyName, batchSubmitter)

		oracleBridgeExecutor, err := weaveio.GenerateKey("init", crypto.CosmosAddressType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate oracle bridge executor mnemonic: %w", err)
		}
		keyFile.AddKey(OracleBridgeExecutorKeyName, oracleBridgeExecutor)

		return keyFile, nil
	case "challenger":
		challenger, err := weaveio.GenerateKey("init", crypto.CosmosAddressType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate challenger mnemonic: %w", err)
		}
		keyFile.AddKey(ChallengerKeyName, challenger)

		return keyFile, nil
	default:
		return nil, fmt.Errorf("unsupported bot name: %s", botName)
	}
}

func buildExecutorConfig(state OPInitBotsState, version int) ExecutorConfig {
	configMap := state.botConfig
	return ExecutorConfig{
		Version: version,
		Server: ServerConfig{
			Address:      configMap["listen_address"],
			AllowOrigins: "*",
			AllowHeaders: "Origin, Content-Type, Accept",
			AllowMethods: "GET",
		},
		L1Node: NodeSettings{
			ChainID:       configMap["l1_node.chain_id"],
			RPCAddress:    configMap["l1_node.rpc_address"],
			Bech32Prefix:  "init",
			GasPrice:      configMap["l1_node.gas_price"],
			GasAdjustment: 1.5,
			TxTimeout:     60,
		},
		L2Node: NodeSettings{
			ChainID:       configMap["l2_node.chain_id"],
			RPCAddress:    configMap["l2_node.rpc_address"],
			Bech32Prefix:  "init",
			GasPrice:      "0" + configMap["l2_node.gas_denom"],
			GasAdjustment: 1.5,
			TxTimeout:     60,
		},
		DANode: NodeSettings{
			ChainID:       configMap["da_node.chain_id"],
			RPCAddress:    configMap["da_node.rpc_address"],
			Bech32Prefix:  configMap["da_node.bech32_prefix"],
			GasPrice:      configMap["da_node.gas_price"],
			GasAdjustment: 1.5,
			TxTimeout:     60,
		},
		BridgeExecutor:                BridgeExecutorKeyName,
		OracleBridgeExecutor:          OracleBridgeExecutorKeyName,
		MaxChunks:                     5000,
		MaxChunkSize:                  300000,
		MaxSubmissionTime:             3600,
		L1StartHeight:                 state.L1StartHeight,
		L2StartHeight:                 1,
		BatchStartHeight:              1,
		DisableDeleteFutureWithdrawal: false,
		DisableAutoSetL1Height:        true,
		DisableBatchSubmitter:         false,
		DisableOutputSubmitter:        false,
	}
}
