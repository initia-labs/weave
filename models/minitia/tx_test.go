package minitia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initia-labs/weave/config"
	weavecontext "github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/cosmosutils"
	"github.com/initia-labs/weave/types"
	"github.com/initia-labs/weave/ui"
	"github.com/spf13/viper"
	"github.com/test-go/testify/require"
)

func daIsolateViper(t *testing.T) {
	t.Helper()
	original := *viper.GetViper()
	*viper.GetViper() = *viper.New()
	t.Cleanup(func() { *viper.GetViper() = original })
	viper.Set("common.analytics_device_id", "da-flow-test")
}

// Catches Initia batch routing through Celestia, missing genesis/funding entries,
// wrong key derivation, or signing against a different L1 chain/RPC.
func TestDAFlowInitiaKeysAndFunding(t *testing.T) {
	for _, generate := range []bool{true, false} {
		t.Run(map[bool]string{true: "generate", false: "recover"}[generate], func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("DYLD_LIBRARY_PATH", os.Getenv("DYLD_LIBRARY_PATH"))
			t.Setenv("LD_LIBRARY_PATH", os.Getenv("LD_LIBRARY_PATH"))
			daIsolateViper(t)
			daRegistryFixture(t)
			mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
			gasKey, err := config.RecoverGasStationKey(mnemonic)
			require.NoError(t, err)
			viper.Set("common.gas_station", gasKey)
			// Fake only the external binary boundary. Real key helpers, funding JSON,
			// config lookup, chain selection, signing/broadcast orchestration remain active.
			callsPath := filepath.Join(home, "calls.txt")
			txPath := filepath.Join(home, "signed-tx.json")
			celestiaCalls := filepath.Join(home, "celestia-called")
			t.Setenv("DA_TEST_CALLS", callsPath)
			t.Setenv("DA_TEST_TX", txPath)
			t.Setenv("DA_TEST_GAS_ADDRESS", gasKey.InitiaAddress)
			script := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$DA_TEST_CALLS"
case "$1 $2" in
 'keys show') exit 1 ;;
 'keys delete') exit 0 ;;
 'keys add')
  read -r input
  key="$3"
  if [ "$key" = weave.DummyKey ]; then
   case "$input" in
    'operator words') key=weave.Operator ;;
    'bridge words') key=weave.BridgeExecutor ;;
    'output words') key=weave.OutputSubmitter ;;
    'batch words') key=weave.BatchSubmitter ;;
    'challenger words') key=weave.Challenger ;;
    *) exit 17 ;;
   esac
  fi
  case "$key" in
   weave.Operator) role=operator ;;
   weave.BridgeExecutor) role=bridge ;;
   weave.OutputSubmitter) role=output ;;
   weave.BatchSubmitter) role=batch ;;
   weave.Challenger) role=challenger ;;
   weave.GasStation) role=gas ;;
   *) exit 18 ;;
  esac
  if [ "$role" = gas ]; then address="$DA_TEST_GAS_ADDRESS";else address="init1$role";fi
  printf '{"name":"%s","type":"local","address":"%s","pubkey":"fixture","mnemonic":"%s words"}\n' "$3" "$address" "$role"
  ;;
 'query bank') printf '%s\n' '{"balances":[{"denom":"uinit","amount":"7000000"}],"pagination":{"next_key":null,"total":"1"}}' ;;
 'tx sign') cp "$3" "$DA_TEST_TX" ;;
 'tx broadcast') printf '%s\n' '{"code":0,"txhash":"INITIA-FUNDED","raw_log":""}' ;;
 'query tx') printf '%s\n' '{"code":0,"txhash":"INITIA-FUNDED","raw_log":""}' ;;
 *) exit 19 ;;
esac
`
			binary := filepath.Join(home, "minimove")
			daWriteFile(t, binary, script, 0700)
			l1Binary, err := cosmosutils.GetInitiaBinaryPath("v1.2.3")
			require.NoError(t, err)
			daWriteFile(t, l1Binary, script, 0700)
			celestiaBinary := filepath.Join(home, "celestia-appd")
			daWriteFile(t, celestiaBinary, "#!/bin/sh\ntouch '"+celestiaCalls+"'\nexit 91\n", 0700)
			osName := map[string]string{"darwin": "Darwin", "linux": "Linux"}[runtime.GOOS]
			arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[runtime.GOARCH]
			oldTransport := http.DefaultTransport
			requests := []string{}
			http.DefaultTransport = daRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests = append(requests, r.URL.String())
				switch r.URL.String() {
				case "https://test-lcd.invalid/cosmos/base/tendermint/v1beta1/node_info":
					return daJSONResponse(r, `{"default_node_info":{"network":"initia-test-1"},"application_version":{"name":"initia","version":"v1.2.3"}}`), nil
				case "https://api.github.com/repos/initia-labs/initia/releases?per_page=100":
					return daJSONResponse(r, fmt.Sprintf(`[{"tag_name":"v1.2.3","prerelease":false,"assets":[{"browser_download_url":"https://binaries.invalid/initia_1.2.3_%s_%s.tar.gz"}]}]`, osName, arch)), nil
				default:
					t.Errorf("unexpected HTTP request: %s", r.URL)
					return daJSONResponse(r, `{}`), nil
				}
			})
			t.Cleanup(func() { http.DefaultTransport = oldTransport })
			state := daLaunchState()
			selector := NewOpBridgeBatchSubmissionTargetSelect(weavecontext.NewAppContext(state))
			for i, option := range selector.Options {
				if option == Initia {
					selector.Cursor = i
					break
				}
			}
			next, _ := selector.Update(tea.KeyMsg{Type: tea.KeyEnter})
			state = weavecontext.GetCurrentState[LaunchState](next.(*OracleEnableSelect).Ctx)
			require.Equal(t, "INITIA", state.opBridgeBatchSubmissionTarget)
			require.False(t, state.batchSubmissionIsCelestia)
			state.generateKeys = generate
			state.binaryPath = binary
			state.celestiaBinaryPath = celestiaBinary
			state.genesisAccounts = nil
			msg := generateOrRecoverSystemKeys(weavecontext.NewAppContext(state))()
			finished, ok := msg.(ui.EndLoading)
			require.True(t, ok, "key generation returned %T: %+v", msg, msg)
			state = weavecontext.GetCurrentState[LaunchState](finished.Ctx)
			require.Equal(t, "init1batch", state.systemKeyBatchSubmitterAddress)
			require.Equal(t, "batch words", state.systemKeyBatchSubmitterMnemonic)
			require.Len(t, state.genesisAccounts, 5)
			batchCount := 0
			for _, acc := range state.genesisAccounts {
				if acc.Address == "init1batch" {
					batchCount++
					require.Equal(t, "0umin", acc.Coins)
				}
			}
			require.Equal(t, 1, batchCount)
			state.FillDefaultBalances()
			keys := NewL1SystemKeys(&types.GenesisAccount{Address: state.systemKeyBridgeExecutorAddress, Coins: state.systemKeyL1BridgeExecutorBalance}, &types.GenesisAccount{Address: state.systemKeyOutputSubmitterAddress, Coins: state.systemKeyL1OutputSubmitterBalance}, &types.GenesisAccount{Address: state.systemKeyBatchSubmitterAddress, Coins: state.systemKeyL1BatchSubmitterBalance}, &types.GenesisAccount{Address: state.systemKeyChallengerAddress, Coins: state.systemKeyL1ChallengerBalance})
			l1Want, daWant, err := keys.calculateTotalWantedCoins(&state)
			require.NoError(t, err)
			require.Equal(t, "7000000", l1Want.String())
			require.Equal(t, "0", daWant.String())
			require.NoError(t, keys.VerifyGasStationBalances(&state))
			fundingMsg := broadcastFundingFromGasStation(weavecontext.NewAppContext(state))()
			fundingEnd, ok := fundingMsg.(ui.EndLoading)
			require.True(t, ok, "funding returned %T: %+v", fundingMsg, fundingMsg)
			fundedState := weavecontext.GetCurrentState[LaunchState](fundingEnd.Ctx)
			require.Empty(t, fundedState.systemKeyCelestiaFundingTxHash)
			require.Equal(t, "INITIA-FUNDED", fundedState.systemKeyL1FundingTxHash)

			bz, err := os.ReadFile(txPath)
			require.NoError(t, err)
			var tx struct {
				Body struct {
					Messages []struct {
						Type   string `json:"@type"`
						From   string `json:"from_address"`
						To     string `json:"to_address"`
						Amount []struct{ Denom, Amount string }
					}
				}
			}
			require.NoError(t, json.Unmarshal(bz, &tx))
			require.Len(t, tx.Body.Messages, 4)
			for i, want := range []struct{ address, amount string }{{"init1bridge", "2000000"}, {"init1output", "2000000"}, {"init1batch", "1000000"}, {"init1challenger", "2000000"}} {
				actual := tx.Body.Messages[i]
				require.Equal(t, "/cosmos.bank.v1beta1.MsgSend", actual.Type)
				require.Equal(t, gasKey.InitiaAddress, actual.From)
				require.Equal(t, want.address, actual.To)
				require.Len(t, actual.Amount, 1)
				require.Equal(t, "uinit", actual.Amount[0].Denom)
				require.Equal(t, want.amount, actual.Amount[0].Amount)
			}
			calls, err := os.ReadFile(callsPath)
			require.NoError(t, err)
			lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
			adds := []string{}
			for _, line := range lines {
				if strings.HasPrefix(line, "keys add ") {
					adds = append(adds, line)
				}
				if strings.HasPrefix(line, "tx sign ") {
					require.Contains(t, line, "--node https://test-rpc.invalid --chain-id initia-test-1 --keyring-backend test")
				}
				if strings.HasPrefix(line, "tx broadcast ") {
					require.Contains(t, line, "--node https://test-rpc.invalid --output json")
				}
			}
			require.Len(t, adds, 6)
			for _, line := range adds[:5] {
				require.Contains(t, line, "--coin-type 118 --key-type secp256k1")
				if !generate {
					require.Contains(t, line, "--recover")
				}
			}
			if generate {
				require.Contains(t, adds[3], "keys add weave.BatchSubmitter ")
			} else {
				require.Contains(t, adds[3], "keys add weave.DummyKey ")
			}
			require.Contains(t, string(calls), "tx sign ")
			require.Contains(t, string(calls), "tx broadcast ")
			require.Contains(t, string(calls), "query bank balances ")
			require.Len(t, requests, 4)
			_, err = os.Stat(celestiaCalls)
			require.True(t, os.IsNotExist(err), "Celestia binary must never run")
		})
	}
}
