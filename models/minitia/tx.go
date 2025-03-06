package minitia

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/config"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/io"
	"github.com/initia-labs/weave/registry"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/types"
)

var ErrInsufficientBalance = errors.New("insufficient balance")

type L1SystemKeys struct {
	srv             service.Service
	celestiaService service.Service
	BridgeExecutor  *types.GenesisAccount
	OutputSubmitter *types.GenesisAccount
	BatchSubmitter  *types.GenesisAccount
	Challenger      *types.GenesisAccount
}

func NewL1SystemKeys(srv service.Service, celestiaService service.Service, bridgeExecutor, outputSubmitter, batchSubmitter, challenger *types.GenesisAccount) *L1SystemKeys {
	return &L1SystemKeys{
		srv:             srv,
		celestiaService: celestiaService,
		BridgeExecutor:  bridgeExecutor,
		OutputSubmitter: outputSubmitter,
		BatchSubmitter:  batchSubmitter,
		Challenger:      challenger,
	}
}

type FundAccountsResponse struct {
	CelestiaTx *cosmosutils.InitiadTxResponse
	InitiaTx   *cosmosutils.InitiadTxResponse
}

func (lsk *L1SystemKeys) FundAccountsWithGasStation(state *LaunchState) (*FundAccountsResponse, error) {
	var resp FundAccountsResponse

	gasStationKey, err := config.GetGasStationKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get gas station key: %v", err)
	}

	_, err = cosmosutils.RecoverKeyFromMnemonic(lsk.srv, common.WeaveGasStationKeyName, gasStationKey.Mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to recover gas station key: %v", err)
	}
	defer func() {
		_ = cosmosutils.DeleteKey(lsk.srv, common.WeaveGasStationKeyName)
	}()

	var rawTxContent string
	if state.batchSubmissionIsCelestia {
		rawTxContent = fmt.Sprintf(
			FundMinitiaAccountsWithoutBatchTxInterface,
			gasStationKey.InitiaAddress,
			lsk.BridgeExecutor.Address,
			lsk.BridgeExecutor.Coins,
			lsk.OutputSubmitter.Address,
			lsk.OutputSubmitter.Coins,
			lsk.Challenger.Address,
			lsk.Challenger.Coins,
		)
		_, err = cosmosutils.RecoverKeyFromMnemonic(lsk.srv, common.WeaveGasStationKeyName, gasStationKey.Mnemonic)
		if err != nil {
			return nil, fmt.Errorf("failed to recover celestia gas station key: %v", err)
		}
		defer func() {
			_ = cosmosutils.DeleteKey(lsk.srv, common.WeaveGasStationKeyName)
		}()

		// TODO: Choose DA layer based on the chosen L1 network
		celestiaRegistry, err := registry.GetChainRegistry(registry.CelestiaTestnet)
		if err != nil {
			return nil, fmt.Errorf("failed to get celestia registry: %v", err)
		}

		celestiaRpc, err := celestiaRegistry.GetActiveRpc()
		if err != nil {
			return nil, fmt.Errorf("failed to get active rpc for celestia: %v", err)
		}

		//celestiaMinGasPrice, err := celestiaRegistry.GetMinGasPriceByDenom(DefaultCelestiaGasDenom)
		//if err != nil {
		//	return nil, fmt.Errorf("failed to get celestia minimum gas price: %v", err)
		//}

		celestiaChainId := celestiaRegistry.GetChainId()
		sendCmd := lsk.celestiaService.RunCmd([]string{"tx", "bank", "send"}, common.WeaveGasStationKeyName,
			lsk.BatchSubmitter.Address, fmt.Sprintf("%sutia", lsk.BatchSubmitter.Coins), "--node", celestiaRpc,
			"--chain-id", celestiaChainId, "--gas", "200000", "--gas-prices", "0.1utia", "--output", "json", "-y",
		)
		broadcastRes, err := sendCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to broadcast transaction: %v", err)
		}

		var txResponse cosmosutils.InitiadTxResponse
		err = json.Unmarshal(broadcastRes, &txResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
		}
		if txResponse.Code != 0 {
			return nil, fmt.Errorf("celestia tx failed with error: %v", txResponse.RawLog)
		}
		err = lsk.waitForTransactionInclusion(celestiaRpc, txResponse.TxHash)
		if err != nil {
			return nil, err
		}
		resp.CelestiaTx = &txResponse
	} else {
		rawTxContent = fmt.Sprintf(
			FundMinitiaAccountsDefaultTxInterface,
			gasStationKey.InitiaAddress,
			lsk.BridgeExecutor.Address,
			lsk.BridgeExecutor.Coins,
			lsk.OutputSubmitter.Address,
			lsk.OutputSubmitter.Coins,
			lsk.BatchSubmitter.Address,
			lsk.BatchSubmitter.Coins,
			lsk.Challenger.Address,
			lsk.Challenger.Coins,
		)
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home: %v", err)
	}
	rawTxPath := filepath.Join(userHome, common.WeaveDataDirectory, TmpTxFilename)
	if err = io.WriteFile(rawTxPath, rawTxContent); err != nil {
		return nil, fmt.Errorf("failed to write raw tx file: %v", err)
	}
	defer func() {
		if err := io.DeleteFile(rawTxPath); err != nil {
			fmt.Printf("failed to delete raw tx file: %v", err)
		}
	}()

	signCmd := exec.Command(state.binaryPath, "tx", "sign", rawTxPath, "--from", common.WeaveGasStationKeyName, "--node", state.l1RPC,
		"--chain-id", state.l1ChainId, "--keyring-backend", "test", "--output-document", rawTxPath)
	signRes, err := signCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v, output: %s", err, string(signRes))
	}

	broadcastCmd := exec.Command(state.binaryPath, "tx", "broadcast", rawTxPath, "--node", state.l1RPC, "--output", "json")
	broadcastRes, err := broadcastCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %v, output: %s", err, string(broadcastRes))
	}

	var txResponse cosmosutils.InitiadTxResponse
	err = json.Unmarshal(broadcastRes, &txResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	if txResponse.Code != 0 {
		return nil, fmt.Errorf("initia l1 tx failed with error: %v", txResponse.RawLog)
	}

	err = lsk.waitForTransactionInclusion(state.l1RPC, txResponse.TxHash)
	if err != nil {
		return nil, err
	}
	resp.InitiaTx = &txResponse

	return &resp, nil
}

// waitForTransactionInclusion polls for the transaction inclusion in a block
func (lsk *L1SystemKeys) waitForTransactionInclusion(rpcURL, txHash string) error {
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
			statusCmd := lsk.celestiaService.RunCmd([]string{"query", "tx", txHash, "--node", rpcURL, "--output", "json"})
			statusRes, err := statusCmd.CombinedOutput()
			// If the transaction is not included in a block yet, just continue polling
			if err != nil {
				// You can add more detailed error handling here if needed,
				// but for now, we continue polling if it returns an error (i.e., "not found" or similar).
				continue
			}

			var txResponse cosmosutils.MinimalTxResponse
			err = json.Unmarshal(statusRes, &txResponse)
			if err != nil {
				return fmt.Errorf("failed to unmarshal transaction JSON response: %v", err)
			}
			if txResponse.Code == 0 { // Successful transaction
				// Transaction successfully included in block
				return nil
			} else {
				return fmt.Errorf("tx failed with error: %v", txResponse.RawLog)
			}

			// If the transaction is not in a block yet, continue polling
		}
	}
}

const FundMinitiaAccountsDefaultTxInterface = `
{
  "body":{
    "messages":[
      {
        "@type":"/cosmos.bank.v1beta1.MsgSend",
        "from_address":"%[1]s",
        "to_address":"%[2]s",
        "amount":[
          {
            "denom":"uinit",
            "amount":"%[3]s"
          }
        ]
      },
      {
        "@type":"/cosmos.bank.v1beta1.MsgSend",
        "from_address":"%[1]s",
        "to_address":"%[4]s",
        "amount":[
          {
            "denom":"uinit",
            "amount":"%[5]s"
          }
        ]
      },
      {
        "@type":"/cosmos.bank.v1beta1.MsgSend",
        "from_address":"%[1]s",
        "to_address":"%[6]s",
        "amount":[
          {
            "denom":"uinit",
            "amount":"%[7]s"
          }
        ]
      },
      {
        "@type":"/cosmos.bank.v1beta1.MsgSend",
        "from_address":"%[1]s",
        "to_address":"%[8]s",
        "amount":[
          {
            "denom":"uinit",
            "amount":"%[9]s"
          }
        ]
      }
    ],
    "memo":"Sent from Weave Gas Station!",
    "timeout_height":"0",
    "extension_options":[],
    "non_critical_extension_options":[]
  },
  "auth_info":{
    "signer_infos":[],
    "fee":{
      "amount":[
        {
          "denom":"uinit",
          "amount":"12000"
        }
      ],
      "gas_limit":"800000",
      "payer":"",
      "granter":""
    },
    "tip":null
  },
  "signatures":[]
}
`

const FundMinitiaAccountsWithoutBatchTxInterface = `
{
  "body":{
    "messages":[
      {
        "@type":"/cosmos.bank.v1beta1.MsgSend",
        "from_address":"%[1]s",
        "to_address":"%[2]s",
        "amount":[
          {
            "denom":"uinit",
            "amount":"%[3]s"
          }
        ]
      },
      {
        "@type":"/cosmos.bank.v1beta1.MsgSend",
        "from_address":"%[1]s",
        "to_address":"%[4]s",
        "amount":[
          {
            "denom":"uinit",
            "amount":"%[5]s"
          }
        ]
      },
      {
        "@type":"/cosmos.bank.v1beta1.MsgSend",
        "from_address":"%[1]s",
        "to_address":"%[6]s",
        "amount":[
          {
            "denom":"uinit",
            "amount":"%[7]s"
          }
        ]
      }
    ],
    "memo":"Sent from Weave Gas Station!",
    "timeout_height":"0",
    "extension_options":[],
    "non_critical_extension_options":[]
  },
  "auth_info":{
    "signer_infos":[],
    "fee":{
      "amount":[
        {
          "denom":"uinit",
          "amount":"10500"
        }
      ],
      "gas_limit":"700000",
      "payer":"",
      "granter":""
    },
    "tip":null
  },
  "signatures":[]
}
`

func (lsk *L1SystemKeys) calculateTotalWantedCoins(state *LaunchState) (l1Want *big.Int, daWant *big.Int, err error) {
	l1Want = new(big.Int)
	daWant = new(big.Int)

	for _, acc := range []*types.GenesisAccount{
		lsk.BridgeExecutor,
		lsk.OutputSubmitter,
		lsk.BatchSubmitter,
		lsk.Challenger,
	} {
		if acc == nil {
			continue
		}

		amount := new(big.Int)
		_, ok := amount.SetString(acc.Coins, 10)
		if !ok {
			return nil, nil, fmt.Errorf("failed to parse coin amount '%s'", acc.Coins)
		}

		if acc == lsk.Challenger && state.batchSubmissionIsCelestia {
			daWant.Add(daWant, amount)
		} else {
			l1Want.Add(l1Want, amount)
		}
	}

	return l1Want, daWant, nil
}

func queryChainBalance(service service.Service, rpc, address string) (map[string]string, error) {
	queryCmd := service.RunCmd([]string{"query", "bank", "balances", address,
		"--node", rpc, "--output", "json"})
	balanceRes, err := queryCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to query balance: %v", err)
	}

	var balance struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(balanceRes, &balance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal balance: %v", err)
	}

	balanceMap := make(map[string]string)
	for _, bal := range balance.Balances {
		balanceMap[bal.Denom] = bal.Amount
	}

	return balanceMap, nil
}

func (lsk *L1SystemKeys) VerifyGasStationBalances(state *LaunchState) error {
	gasStationKey, err := config.GetGasStationKey()
	if err != nil {
		return fmt.Errorf("failed to get gas station key: %v", err)
	}

	// Query L1 balances
	l1Balances, err := queryChainBalance(lsk.srv, state.l1RPC, gasStationKey.InitiaAddress)
	if err != nil {
		return fmt.Errorf("failed to query L1 balance: %v", err)
	}

	// Calculate required balances
	l1Want, daWant, err := lsk.calculateTotalWantedCoins(state)
	if err != nil {
		return fmt.Errorf("failed to calculate wanted coins: %v", err)
	}

	// Verify L1 balance
	l1AvailableBig := new(big.Int)
	if _, ok := l1AvailableBig.SetString(l1Balances[DefaultL1GasDenom], 10); !ok {
		return fmt.Errorf("failed to parse L1 available balance: %s", l1Balances[DefaultL1GasDenom])
	}

	if l1AvailableBig.Cmp(l1Want) < 0 {
		return fmt.Errorf("%w: insufficient initia balance: have %s uinit, want %s uinit",
			ErrInsufficientBalance, l1AvailableBig.String(), l1Want.String())
	}

	// Check Celestia balance if needed
	if state.batchSubmissionIsCelestia {
		if err := lsk.verifyCelestiaBalance(state, daWant); err != nil {
			return err
		}
	}

	return nil
}

func (lsk *L1SystemKeys) verifyCelestiaBalance(state *LaunchState, daWant *big.Int) error {
	gasStationKey, err := config.GetGasStationKey()
	if err != nil {
		return fmt.Errorf("failed to get gas station key: %v", err)
	}

	// TODO: Choose DA layer based on the chosen L1 network
	celestiaRegistry, err := registry.GetChainRegistry(registry.CelestiaTestnet)
	if err != nil {
		return fmt.Errorf("failed to get celestia registry: %v", err)
	}

	celestiaRpc, err := celestiaRegistry.GetActiveRpc()
	if err != nil {
		return fmt.Errorf("failed to get active rpc for celestia: %v", err)
	}

	// Query Celestia balances
	celestiaBalances, err := queryChainBalance(state.celestiaService, celestiaRpc, gasStationKey.CelestiaAddress)
	if err != nil {
		return fmt.Errorf("failed to query Celestia balance: %v", err)
	}

	// Verify Celestia balance
	daAvailableBig := new(big.Int)
	if _, ok := daAvailableBig.SetString(celestiaBalances[DefaultCelestiaGasDenom], 10); !ok {
		return fmt.Errorf("failed to parse DA available balance: %s", celestiaBalances[DefaultCelestiaGasDenom])
	}

	if daAvailableBig.Cmp(daWant) < 0 {
		return fmt.Errorf("insufficient DA balance. Required: %s%s, Available: %s%s",
			daWant.String(), DefaultCelestiaGasDenom,
			daAvailableBig.String(), DefaultCelestiaGasDenom)
	}

	return nil
}
