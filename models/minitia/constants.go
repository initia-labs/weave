package minitia

import "math/big"

const (
	AppName string = "minitiad"

	OperatorKeyName        string = "weave.Operator"
	BridgeExecutorKeyName  string = "weave.BridgeExecutor"
	OutputSubmitterKeyName string = "weave.OutputSubmitter"
	BatchSubmitterKeyName  string = "weave.BatchSubmitter"
	ChallengerKeyName      string = "weave.Challenger"

	DefaultL1BridgeExecutorBalance  string = "2000000"
	DefaultL1OutputSubmitterBalance string = "2000000"
	DefaultL1BatchSubmitterBalance  string = "1000000"
	DefaultL1ChallengerBalance      string = "2000000"
	DefaultL2BridgeExecutorBalance  string = "100000000"

	TmpTxFilename string = "weave.minitia.tx.json"

	DefaultL1GasDenom       string = "uinit"
	DefaultL1GasPrices             = "0.15" + DefaultL1GasDenom
	DefaultCelestiaGasDenom string = "utia"

	MaxMonikerLength int = 70
	MaxChainIDLength int = 50

	LaunchConfigFilename = "minitia.config.json"

	CelestiaAppName string = "celestia-appd"

	InitiaScanURL         string = "https://scan.testnet.initia.xyz"
	DefaultMinitiaLCD     string = "http://localhost:1317"
	DefaultMinitiaRPC     string = "http://localhost:26657"
	DefaultMinitiaJsonRPC string = "http://localhost:8545"
)

var (
	DefaultL1InitiaNeededBalanceIfCelestiaDA string
	DefaultL1InitiaNeededBalanceIfInitiaDA   string
)

func init() {
	total := big.NewInt(0)
	values := []string{
		DefaultL1BridgeExecutorBalance,
		DefaultL1OutputSubmitterBalance,
		DefaultL1ChallengerBalance,
	}

	for _, v := range values {
		num := new(big.Int)
		num, _ = num.SetString(v, 10)
		total.Add(total, num)
	}

	DefaultL1InitiaNeededBalanceIfCelestiaDA = total.String()

	num := new(big.Int)
	num, _ = num.SetString(DefaultL1BatchSubmitterBalance, 10)
	total.Add(total, num)
	DefaultL1InitiaNeededBalanceIfInitiaDA = total.String()
}
