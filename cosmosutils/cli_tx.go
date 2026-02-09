package cosmosutils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/initia-labs/weave/client"
)

const (
	DefaultGasAdjustment = "1.4"
	TmpKeyName           = "weave.clitx.executor"
)

type InitiadTxResponse struct {
	Height    string         `json:"height"`
	TxHash    string         `json:"txhash"`
	Codespace string         `json:"codespace"`
	Code      int            `json:"code"`
	Data      string         `json:"data"`
	RawLog    string         `json:"raw_log"`
	Logs      *[]interface{} `json:"logs"`
	Info      string         `json:"info"`
	GasWanted string         `json:"gas_wanted"`
	GasUsed   string         `json:"gas_used"`
	Tx        *[]interface{} `json:"tx"`
	Timestamp string         `json:"timestamp"`
	Events    *[]interface{} `json:"events"`
}

type MinimalTxResponse struct {
	Code   int    `json:"code"`
	RawLog string `json:"raw_log"`
}

type MinimalRPCTxResponse struct {
	Result struct {
		TxResult struct {
			Code int    `json:"code"`
			Log  string `json:"log"`
		} `json:"tx_result"`
	} `json:"result"`
}

type InitiadTxExecutor struct {
	binaryPath string
}

type MinitiadTxExecutor struct {
	binaryPath string
}

type MsgUpdateParams struct {
	Authority string        `json:"authority"`
	Params    OPChildParams `json:"params"`
}

type Message struct {
	Type      string        `json:"@type"`
	Authority string        `json:"authority"`
	Params    OPChildParams `json:"params"`
}

type JsonPayload struct {
	Messages []Message `json:"messages"`
}

func CreateOPChildUpdateParamsMsg(filePath string, params OPChildParams) error {
	message := Message{
		Type:      "/opinit.opchild.v1.MsgUpdateParams",
		Authority: "init1gz9n8jnu9fgqw7vem9ud67gqjk5q4m2w0aejne",
		Params:    params,
	}

	jsonPayload := JsonPayload{
		Messages: []Message{message},
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(jsonPayload, "", "    ")
	if err != nil {
		return err
	}

	// Create or open the file where the data will be saved
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Write the JSON data to the file
	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}
	return nil
}

func NewInitiadTxExecutor(rest string) (*InitiadTxExecutor, error) {
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

	return &InitiadTxExecutor{
		binaryPath: binaryPath,
	}, nil
}

func (te *InitiadTxExecutor) BroadcastMsgSend(senderMnemonic, recipientAddress, amount, gasPrices, rpc, chainId string, coinType int) (*InitiadTxResponse, error) {
	return broadcastMsgSend(te.binaryPath, senderMnemonic, recipientAddress, amount, gasPrices, rpc, chainId, coinType)
}

func NewMinitiadTxExecutor(rest string) (*MinitiadTxExecutor, error) {
	httpClient := client.NewHTTPClient()
	vm, version, url, err := GetMinitiadBinaryUrlFromLcd(httpClient, rest)
	if err != nil {
		return nil, err
	}
	binaryPath, err := GetMinitiadBinaryPath(vm, version)
	if err != nil {
		return nil, err
	}
	err = InstallMinitiadBinary(vm, version, url, binaryPath)
	if err != nil {
		return nil, err
	}

	return &MinitiadTxExecutor{binaryPath: binaryPath}, nil
}

func (te *MinitiadTxExecutor) BroadcastMsgSend(senderMnemonic, recipientAddress, amount, gasPrices, rpc, chainId string, coinType int) (*InitiadTxResponse, error) {
	return broadcastMsgSend(te.binaryPath, senderMnemonic, recipientAddress, amount, gasPrices, rpc, chainId, coinType)
}

func broadcastMsgSend(binaryPath, senderMnemonic, recipientAddress, amount, gasPrices, rpc, chainId string, coinType int) (*InitiadTxResponse, error) {
	if coinType == 0 {
		return nil, fmt.Errorf("coin type must be explicitly provided (60 or 118)")
	}
	
	_, err := RecoverKeyFromMnemonicWithCoinType(binaryPath, TmpKeyName, senderMnemonic, coinType)
	if err != nil {
		return nil, fmt.Errorf("failed to recover gas station key: %v", err)
	}
	defer func() {
		_ = DeleteKey(binaryPath, TmpKeyName)
	}()

	cmd := exec.Command(binaryPath, "tx", "bank", "send", TmpKeyName, recipientAddress, amount, "--from",
		TmpKeyName, "--chain-id", chainId, "--gas", "auto", "--gas-adjustment", DefaultGasAdjustment,
		"--gas-prices", gasPrices, "--node", rpc, "--output", "json", "--keyring-backend", "test", "-y")

	outputBytes, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to send tx MsgSend for %s: %v, output: %s", TmpKeyName, err, string(outputBytes))
	}

	var txResponse InitiadTxResponse
	err = json.Unmarshal(outputBytes, &txResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if txResponse.Code != 0 {
		return nil, fmt.Errorf("tx failed with error: %v", txResponse.RawLog)
	}

	err = waitForTransactionInclusion(binaryPath, rpc, txResponse.TxHash)
	if err != nil {
		return nil, err
	}

	return &txResponse, nil
}

func waitForTransactionInclusion(binaryPath, rpcURL, txHash string) error {
	// Poll for transaction status until it's included in a block
	timeout := time.After(15 * time.Second)   // Example timeout for polling
	ticker := time.NewTicker(3 * time.Second) // Poll every 3 seconds
	defer ticker.Stop()                       // Ensure cleanup of ticker resource

	for {
		select {
		case <-timeout:
			return fmt.Errorf("transaction not included in block within timeout")
		case <-ticker.C:
			// Query transaction status
			statusCmd := exec.Command(binaryPath, "query", "tx", txHash, "--node", rpcURL, "--output", "json")
			statusRes, err := statusCmd.CombinedOutput()
			// If the transaction is not included in a block yet, just continue polling
			if err != nil {
				// You can add more detailed error handling here if needed,
				// but for now, we continue polling if it returns an error (i.e., "not found" or similar).
				continue
			}

			var txResponse MinimalTxResponse
			err = json.Unmarshal(statusRes, &txResponse)
			if err != nil {
				return fmt.Errorf("failed to unmarshal transaction JSON response: %v", err)
			}
			if txResponse.Code == 0 { // Successful transaction
				return nil
			} else {
				return fmt.Errorf("tx failed with error: %v", txResponse.RawLog)
			}

			// If the transaction is not in a block yet, continue polling
		}
	}
}
