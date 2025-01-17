package minitia

import (
	"fmt"

	"github.com/initia-labs/weave/types"
)

type LaunchState struct {
	weave              types.WeaveState
	existingMinitiaApp bool
	l1ChainId          string
	l1RPC              string
	vmType             string
	minitiadVersion    string
	minitiadEndpoint   string
	chainId            string
	gasDenom           string
	moniker            string
	enableOracle       bool
	genesisAccounts    types.GenesisAccounts

	opBridgeSubmissionInterval       string
	opBridgeOutputFinalizationPeriod string
	opBridgeBatchSubmissionTarget    string
	batchSubmissionIsCelestia        bool

	generateKeys                     bool
	systemKeyOperatorMnemonic        string
	systemKeyBridgeExecutorMnemonic  string
	systemKeyOutputSubmitterMnemonic string
	systemKeyBatchSubmitterMnemonic  string
	systemKeyChallengerMnemonic      string

	systemKeyOperatorAddress        string
	systemKeyBridgeExecutorAddress  string
	systemKeyOutputSubmitterAddress string
	systemKeyBatchSubmitterAddress  string
	systemKeyChallengerAddress      string

	systemKeyL1BridgeExecutorBalance  string
	systemKeyL1OutputSubmitterBalance string
	systemKeyL1BatchSubmitterBalance  string
	systemKeyL1ChallengerBalance      string
	systemKeyL1FundingTxHash          string
	systemKeyCelestiaFundingTxHash    string

	systemKeyL2OperatorBalance       string
	systemKeyL2BridgeExecutorBalance string

	gasStationExist             bool
	downloadedNewBinary         bool
	downloadedNewCelestiaBinary bool

	preGenesisAccountsResponsesCount int
	preL1BalancesResponsesCount      int
	preL2BalancesResponsesCount      int

	binaryPath         string
	celestiaBinaryPath string

	launchFromExistingConfig bool
	existingConfigPath       string

	feeWhitelistAccounts string
	scanLink             string
}

func (ls LaunchState) Clone() LaunchState {
	clone := LaunchState{
		weave:                             ls.weave,
		existingMinitiaApp:                ls.existingMinitiaApp,
		l1ChainId:                         ls.l1ChainId,
		l1RPC:                             ls.l1RPC,
		vmType:                            ls.vmType,
		minitiadVersion:                   ls.minitiadVersion,
		minitiadEndpoint:                  ls.minitiadEndpoint,
		chainId:                           ls.chainId,
		gasDenom:                          ls.gasDenom,
		moniker:                           ls.moniker,
		enableOracle:                      ls.enableOracle,
		genesisAccounts:                   make(types.GenesisAccounts, len(ls.genesisAccounts)),
		opBridgeSubmissionInterval:        ls.opBridgeSubmissionInterval,
		opBridgeOutputFinalizationPeriod:  ls.opBridgeOutputFinalizationPeriod,
		opBridgeBatchSubmissionTarget:     ls.opBridgeBatchSubmissionTarget,
		batchSubmissionIsCelestia:         ls.batchSubmissionIsCelestia,
		generateKeys:                      ls.generateKeys,
		systemKeyOperatorMnemonic:         ls.systemKeyOperatorMnemonic,
		systemKeyBridgeExecutorMnemonic:   ls.systemKeyBridgeExecutorMnemonic,
		systemKeyOutputSubmitterMnemonic:  ls.systemKeyOutputSubmitterMnemonic,
		systemKeyBatchSubmitterMnemonic:   ls.systemKeyBatchSubmitterMnemonic,
		systemKeyChallengerMnemonic:       ls.systemKeyChallengerMnemonic,
		systemKeyOperatorAddress:          ls.systemKeyOperatorAddress,
		systemKeyBridgeExecutorAddress:    ls.systemKeyBridgeExecutorAddress,
		systemKeyOutputSubmitterAddress:   ls.systemKeyOutputSubmitterAddress,
		systemKeyBatchSubmitterAddress:    ls.systemKeyBatchSubmitterAddress,
		systemKeyChallengerAddress:        ls.systemKeyChallengerAddress,
		systemKeyL1BridgeExecutorBalance:  ls.systemKeyL1BridgeExecutorBalance,
		systemKeyL1OutputSubmitterBalance: ls.systemKeyL1OutputSubmitterBalance,
		systemKeyL1BatchSubmitterBalance:  ls.systemKeyL1BatchSubmitterBalance,
		systemKeyL1ChallengerBalance:      ls.systemKeyL1ChallengerBalance,
		systemKeyL1FundingTxHash:          ls.systemKeyL1FundingTxHash,
		systemKeyCelestiaFundingTxHash:    ls.systemKeyCelestiaFundingTxHash,
		systemKeyL2OperatorBalance:        ls.systemKeyL2OperatorBalance,
		systemKeyL2BridgeExecutorBalance:  ls.systemKeyL2BridgeExecutorBalance,
		gasStationExist:                   ls.gasStationExist,
		downloadedNewBinary:               ls.downloadedNewBinary,
		downloadedNewCelestiaBinary:       ls.downloadedNewCelestiaBinary,
		preGenesisAccountsResponsesCount:  ls.preGenesisAccountsResponsesCount,
		preL1BalancesResponsesCount:       ls.preL1BalancesResponsesCount,
		preL2BalancesResponsesCount:       ls.preL2BalancesResponsesCount,
		binaryPath:                        ls.binaryPath,
		celestiaBinaryPath:                ls.celestiaBinaryPath,
		launchFromExistingConfig:          ls.launchFromExistingConfig,
		existingConfigPath:                ls.existingConfigPath,
		feeWhitelistAccounts:              ls.feeWhitelistAccounts,
		scanLink:                          ls.scanLink,
	}

	copy(clone.genesisAccounts, ls.genesisAccounts)

	return clone
}

func NewLaunchState() *LaunchState {
	return &LaunchState{
		weave: types.NewWeaveState(),
	}
}

func (ls *LaunchState) FillDefaultBalances() {
	ls.systemKeyL1BridgeExecutorBalance = DefaultL1BridgeExecutorBalance
	ls.systemKeyL1OutputSubmitterBalance = DefaultL1OutputSubmitterBalance
	ls.systemKeyL1BatchSubmitterBalance = DefaultL1BatchSubmitterBalance
	ls.systemKeyL1ChallengerBalance = DefaultL1ChallengerBalance
	ls.systemKeyL2BridgeExecutorBalance = fmt.Sprintf("%s%s", DefaultL2BridgeExecutorBalance, ls.gasDenom)
}

func (ls *LaunchState) FinalizeGenesisAccounts() {
	emptyCoins := fmt.Sprintf("0%s", ls.gasDenom)
	accounts := []types.GenesisAccount{
		{Address: ls.systemKeyOperatorAddress, Coins: ls.systemKeyL2OperatorBalance},
		{Address: ls.systemKeyBridgeExecutorAddress, Coins: ls.systemKeyL2BridgeExecutorBalance},
		{Address: ls.systemKeyOutputSubmitterAddress, Coins: emptyCoins},
		{Address: ls.systemKeyChallengerAddress, Coins: emptyCoins},
	}

	if !ls.batchSubmissionIsCelestia {
		accounts = append(accounts, types.GenesisAccount{
			Address: ls.systemKeyBatchSubmitterAddress,
			Coins:   emptyCoins,
		})
	}

	ls.genesisAccounts = append(ls.genesisAccounts, accounts...)
}

func (ls *LaunchState) PrepareLaunchingWithConfig(vm, minitiadVersion, minitiadEndpoint, configPath string, config *types.MinitiaConfig) {
	vmType, err := ParseVMType(vm)
	if err != nil {
		panic(err)
	}
	ls.vmType = string(vmType)
	ls.minitiadVersion = minitiadVersion
	ls.minitiadEndpoint = minitiadEndpoint
	ls.launchFromExistingConfig = true
	ls.existingConfigPath = configPath
	ls.chainId = config.L2Config.ChainID
	ls.gasDenom = config.L2Config.Denom
}
