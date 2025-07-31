package types

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

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

type Metadata struct {
	PermChannels []Channel `json:"perm_channels"`
}

func DecodeBridgeMetadata(base64Str string) (Metadata, error) {
	// Decode the Base64 string
	jsonData, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return Metadata{}, err
	}

	// Struct to hold the decoded JSON
	var metadata Metadata

	// Unmarshal the JSON into the struct
	err = json.Unmarshal(jsonData, &metadata)
	if err != nil {
		fmt.Printf("Error decoding JSON: %v %s\n", err, base64Str)
		return Metadata{}, err
	}

	return metadata, nil
}
