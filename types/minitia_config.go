package types

type MinitiaConfig struct {
	L1Config        *L1Config        `json:"l1_config,omitempty"`
	L2Config        *L2Config        `json:"l2_config,omitempty"`
	OpBridge        *OpBridge        `json:"op_bridge,omitempty"`
	SystemKeys      *SystemKeys      `json:"system_keys,omitempty"`
	GenesisAccounts *GenesisAccounts `json:"genesis_accounts,omitempty"`
}

type L1Config struct {
	ChainID   string `json:"chain_id,omitempty"`
	RpcUrl    string `json:"rpc_url,omitempty"`
	GasPrices string `json:"gas_prices,omitempty"`
}

type L2Config struct {
	ChainID  string `json:"chain_id,omitempty"`
	Denom    string `json:"denom,omitempty"`
	Moniker  string `json:"moniker,omitempty"`
	BridgeID uint64 `json:"bridge_id,omitempty"`
}

type OpBridge struct {
	OutputSubmissionInterval    string `json:"output_submission_interval,omitempty"`
	OutputFinalizationPeriod    string `json:"output_finalization_period,omitempty"`
	OutputSubmissionStartHeight uint64 `json:"output_submission_start_height,omitempty"`
	BatchSubmissionTarget       string `json:"batch_submission_target"`
	EnableOracle                bool   `json:"enable_oracle"`
}

type SystemAccount struct {
	L1Address string `json:"l1_address,omitempty"`
	L2Address string `json:"l2_address,omitempty"`
	DAAddress string `json:"da_address,omitempty"`
	Mnemonic  string `json:"mnemonic,omitempty"`
}

func NewSystemAccount(mnemonic, addresses string) *SystemAccount {
	account := &SystemAccount{
		Mnemonic:  mnemonic,
		L1Address: addresses,
		L2Address: addresses,
	}

	return account
}

func NewBatchSubmitterAccount(mnemonic, address string) *SystemAccount {
	account := &SystemAccount{
		DAAddress: address,
		Mnemonic:  mnemonic,
	}

	return account
}

type GenesisAccount struct {
	Address string `json:"address,omitempty"`
	Coins   string `json:"coins,omitempty"`
}

type GenesisAccounts []GenesisAccount

type SystemKeys struct {
	Validator       *SystemAccount `json:"validator,omitempty"`
	BridgeExecutor  *SystemAccount `json:"bridge_executor,omitempty"`
	OutputSubmitter *SystemAccount `json:"output_submitter,omitempty"`
	BatchSubmitter  *SystemAccount `json:"batch_submitter,omitempty"`
	Challenger      *SystemAccount `json:"challenger,omitempty"`
}

// Artifacts define the structure for the JSON data
type Artifacts struct {
	BridgeID                string `json:"BRIDGE_ID"`
	ExecutorL1MonitorHeight string `json:"EXECUTOR_L1_MONITOR_HEIGHT"`
	ExecutorL2MonitorHeight string `json:"EXECUTOR_L2_MONITOR_HEIGHT"`
}

// Clone returns a deep copy of MinitiaConfig.
// Returns nil if the receiver is nil.
func (m *MinitiaConfig) Clone() *MinitiaConfig {
	if m == nil {
		return nil
	}

	clone := &MinitiaConfig{
		L1Config:        nil,
		L2Config:        nil,
		OpBridge:        nil,
		SystemKeys:      nil,
		GenesisAccounts: nil,
	}

	if m.L1Config != nil {
		clone.L1Config = &L1Config{
			ChainID:   m.L1Config.ChainID,
			RpcUrl:    m.L1Config.RpcUrl,
			GasPrices: m.L1Config.GasPrices,
		}
	}
	// Similar deep copy for other fields...
	if m.L2Config != nil {
		clone.L2Config = &L2Config{
			ChainID:  m.L2Config.ChainID,
			Denom:    m.L2Config.Denom,
			Moniker:  m.L2Config.Moniker,
			BridgeID: m.L2Config.BridgeID,
		}
	}

	if m.OpBridge != nil {
		clone.OpBridge = &OpBridge{
			OutputSubmissionInterval:    m.OpBridge.OutputSubmissionInterval,
			OutputFinalizationPeriod:    m.OpBridge.OutputFinalizationPeriod,
			OutputSubmissionStartHeight: m.OpBridge.OutputSubmissionStartHeight,
			BatchSubmissionTarget:       m.OpBridge.BatchSubmissionTarget,
			EnableOracle:                m.OpBridge.EnableOracle,
		}
	}

	if m.SystemKeys != nil {
		clone.SystemKeys = &SystemKeys{
			Validator:       cloneSystemAccount(m.SystemKeys.Validator),
			BridgeExecutor:  cloneSystemAccount(m.SystemKeys.BridgeExecutor),
			OutputSubmitter: cloneSystemAccount(m.SystemKeys.OutputSubmitter),
			BatchSubmitter:  cloneSystemAccount(m.SystemKeys.BatchSubmitter),
			Challenger:      cloneSystemAccount(m.SystemKeys.Challenger),
		}
	}
	if m.GenesisAccounts != nil {
		accs := make(GenesisAccounts, len(*m.GenesisAccounts))
		copy(accs, *m.GenesisAccounts)
		clone.GenesisAccounts = &accs
	}
	return clone
}

func cloneSystemAccount(acc *SystemAccount) *SystemAccount {
	if acc == nil {
		return nil
	}
	return &SystemAccount{
		L1Address: acc.L1Address,
		L2Address: acc.L2Address,
		DAAddress: acc.DAAddress,
		Mnemonic:  acc.Mnemonic,
	}
}
