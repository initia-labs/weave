package cosmosutils

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/initia-labs/weave/client"
)

type InitiadBankBalancesQueryResponse struct {
	Balances Coins `json:"balances"`
}

type InitiadQuerier struct {
	binaryPath string
}

func NewInitiadQuerier(rest string) *InitiadQuerier {
	httpClient := client.NewHTTPClient()
	nodeVersion, url := MustGetInitiaBinaryUrlFromLcd(httpClient, rest)
	binaryPath := GetInitiaBinaryPath(nodeVersion)
	MustInstallInitiaBinary(nodeVersion, url, binaryPath)

	return &InitiadQuerier{
		binaryPath: binaryPath,
	}
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
		panic(fmt.Sprintf("failed to unmarshal JSON: %v", err))
	}

	return &queryResponse.Balances, nil
}
