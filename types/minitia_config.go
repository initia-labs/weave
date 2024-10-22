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
	Mnemonic  string `json:"mnemonic,omitempty"`
}

func NewSystemAccount(mnemonic string, addresses ...string) *SystemAccount {
	if len(addresses) < 1 || len(addresses) > 2 {
		panic("NewSystemAccount requires 2 or 3 parameters: mnemonic and 1 or 2 addresses")
	}

	account := &SystemAccount{
		Mnemonic:  mnemonic,
		L1Address: addresses[0],
		L2Address: addresses[0],
	}

	if len(addresses) == 2 {
		account.L2Address = addresses[1]
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
