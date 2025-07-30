package types

type BatchInfo struct {
	Submitter string `json:"submitter"`
	ChainType string `json:"chain_type"`
}

type BridgeConfig struct {
	Challenger            string    `json:"challenger"`
	Proposer              string    `json:"proposer"`
	BatchInfo             BatchInfo `json:"batch_info"`
	SubmissionInterval    string    `json:"submission_interval"`
	FinalizationPeriod    string    `json:"finalization_period"`
	SubmissionStartHeight string    `json:"submission_start_height"`
	OracleEnabled         bool      `json:"oracle_enabled"`
	Metadata              string    `json:"metadata"`
}

type Bridge struct {
	BridgeID     string       `json:"bridge_id"`
	BridgeAddr   string       `json:"bridge_addr"`
	BridgeConfig BridgeConfig `json:"bridge_config"`
}
