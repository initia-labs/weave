package cosmosutils

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/types"
)

const (
	DefaultMinitiadQuerierAppName string = "minitiad"
	DefaultMinitiadQuerierVM      string = "evm"
)

type InitiadQuerier struct {
	binaryPath string
}

func NewInitiadQuerier(rest string) (*InitiadQuerier, error) {
	httpClient := client.NewHTTPClient()
	nodeVersion, url, err := GetInitiaBinaryUrlFromLcd(httpClient, rest)
	if err != nil {
		return nil, err
	}
	binaryPath, err := GetInitiaBinaryPath(nodeVersion)
	if err != nil {
		return nil, err
	}
	err = InstallInitiaBinary(nodeVersion, url, binaryPath)
	if err != nil {
		return nil, err
	}

	return &InitiadQuerier{
		binaryPath: binaryPath,
	}, nil
}

type InitiadBankBalancesQueryResponse struct {
	Balances Coins `json:"balances"`
}

func (iq *InitiadQuerier) QueryBankBalances(address, rpc string) (*Coins, error) {
	cmd := exec.Command(iq.binaryPath, "query", "bank", "balances", address, "--node", rpc, "--output", "json")

	outputBytes, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query bank balances for %s: %v, output: %s", address, err, string(outputBytes))
	}

	var queryResponse InitiadBankBalancesQueryResponse
	err = json.Unmarshal(outputBytes, &queryResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return &queryResponse.Balances, nil
}

type MinitiadQuerier struct {
	binaryPath string
}

func NewMinitiadQuerier() (*MinitiadQuerier, error) {
	version, downloadURL, err := GetLatestMinitiaVersion(DefaultMinitiadQuerierVM)
	if err != nil {
		return nil, err
	}

	binaryPath, err := EnsureMinitiadBinary(DefaultMinitiadQuerierVM, version, downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure minitiad binary: %v", err)
	}

	return &MinitiadQuerier{binaryPath: binaryPath}, nil
}

type OPChildBridgeInfoQueryResponse struct {
	BridgeInfo struct {
		BridgeId string `json:"bridge_id"`
	} `json:"bridge_info"`
}

func (mq *MinitiadQuerier) QueryOPChildBridgeInfo(rpc string) (*types.Bridge, error) {
	cmd := exec.Command(mq.binaryPath, "query", "opchild", "bridge-info", "--node", rpc, "--output", "json")

	outputBytes, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query bridge info for %s: %v, output: %s", rpc, err, string(outputBytes))
	}

	var queryResponse OPChildBridgeInfoQueryResponse
	err = json.Unmarshal(outputBytes, &queryResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return &types.Bridge{BridgeID: queryResponse.BridgeInfo.BridgeId}, nil
}

type OPChildNextL1SequenceResponse struct {
	NextL1Sequence string `json:"next_l1_sequence"`
}

func (mq *MinitiadQuerier) QueryOPChildNextL1Sequence(rpc string) (string, error) {
	cmd := exec.Command(mq.binaryPath, "query", "opchild", "next-l1-sequence", "--node", rpc, "--output", "json")

	outputBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query next l1 sequence for %s: %v, output: %s", rpc, err, string(outputBytes))
	}

	var queryResponse OPChildNextL1SequenceResponse
	err = json.Unmarshal(outputBytes, &queryResponse)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return queryResponse.NextL1Sequence, nil
}
