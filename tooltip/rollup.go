package tooltip

import (
	"github.com/initia-labs/weave/ui"
)

var (
	RollupChainIdTooltip      = ui.NewTooltip("Rollup chain ID", ChainIDDescription("rollup"), "", []string{}, []string{}, []string{})
	RollupRPCEndpointTooltip  = ui.NewTooltip("Rollup RPC endpoint", RPCEndpointDescription("rollup"), "", []string{}, []string{}, []string{})
	RollupRESTEndpointTooltip = ui.NewTooltip("Rollup REST endpoint", RESTEndpointDescription("rollup"), "", []string{}, []string{}, []string{})
	RollupGasDenomTooltip     = ui.NewTooltip("Rollup gas denom", GasDenomDescription("rollup"), "", []string{}, []string{}, []string{})
	RollupGasPriceTooltip     = ui.NewTooltip("Rollup gas price", GasPriceDescription("rollup"), "", []string{}, []string{}, []string{})

	// OP Bridge Tooltips
	OpBridgeSubmissionIntervalTooltip       = ui.NewTooltip("Submission Interval", "The interval at which to submit the rollup output root to Initia L1.", "", []string{}, []string{}, []string{})
	OpBridgeOutputFinalizationPeriodTooltip = ui.NewTooltip("Output Finalization Period", "The time period during which submitted output roots can be challenged before being considered final. After this period, the output becomes immutable.", "", []string{}, []string{}, []string{})
	OpBridgeBatchSubmissionTargetTooltip    = ui.NewTooltip("Batch Submission Target", "The target chain for submitting rollup blocks and transaction data to ensure Data Availability. Currently, submissions can be made to Initia L1 or Celestia.\n\nFor production use, we recommend Celestia.", "", []string{}, []string{}, []string{})
	EnableOracleTooltip                     = ui.NewTooltip("Oracle", "Enabling the Oracle feature allows the rollup and contracts deployed on the rollup to access asset price data relayed from the Initia L1.", "", []string{}, []string{}, []string{})

	// System Key Tooltips
	SystemKeyOperatorMnemonicTooltip        = ui.NewTooltip("Rollup Operator", "The operator, also known as Sequencer, is responsible for creating blocks, ordering and including transactions within each block, and maintaining the operation of the rollup network.", "", []string{}, []string{}, []string{})
	SystemKeyBridgeExecutorMnemonicTooltip  = ui.NewTooltip("Bridge Executor", "The executor monitors the L1 and rollup transactions, facilitates token bridging and withdrawals between the rollup and Initia L1 chain, and also relays oracle price feed to rollup.", "", []string{}, []string{}, []string{})
	SystemKeyOutputSubmitterMnemonicTooltip = ui.NewTooltip("Output Submitter", "The submitter submits rollup output roots to L1 for verification and potential challenges. If the submitted output remains unchallenged beyond the output finalization period, it is considered finalized and immutable.", "", []string{}, []string{}, []string{})
	SystemKeyBatchSubmitterMnemonicTooltip  = ui.NewTooltip("Batch Submitter", "The batch submitter submits block and transactions data in batches into a chain to ensure Data Availability. Currently, submissions can be made to Initia L1 or Celestia.", "", []string{}, []string{}, []string{})
	SystemKeyChallengerMnemonicTooltip      = ui.NewTooltip("Challenger", "The challenger prevents misconduct and invalid rollup state submissions by monitoring for output proposals and challenging any that are invalid.", "", []string{}, []string{}, []string{})

	// System Accounts funding
	SystemAccountsFundingPresetTooltip = ui.NewTooltip(
		"OPInit Bots",
		"Bridge Executor: Monitors the L1 and rollup transactions, facilitates token bridging and withdrawals between the rollup and Initia L1 chain, and also relays oracle price feed to rollup.\n\nOutput Submitter: Submits rollup output roots to L1 for verification and potential challenges. If the submitted output remains unchallenged beyond the output finalization period, it is considered finalized and immutable.\n\nBatch Submitter: Submits block and transactions data in batches into a chain to ensure Data Availability. Currently, submissions can be made to Initia L1 or Celestia.\n\nChallenger: Prevents misconduct and invalid rollup state submissions by monitoring for output proposals and challenging any that are invalid.\n\nRollup Operator: Also known as Sequencer, is responsible for creating blocks, ordering and including transactions within each block, and maintaining the operation of the rollup network.",
		"", []string{"Bridge Executor:", "Output Submitter:", "Batch Submitter:", "Challenger:", "Rollup Operator:"}, []string{}, []string{},
	)
	GasStationInRollupGenesisTooltip        = ui.NewTooltip("Gas station in rollup genesis", "Adding gas station account as a genesis account grants initial balances to the gas station on the rollup at network launch. This ensures the gas station can fund relayer accounts during the Weave relayer flow, eliminating the need for later manual funding.", "", []string{}, []string{}, []string{})
	GasStationBalanceOnRollupGenesisTooltip = ui.NewTooltip("Gas station genesis balance", "A genesis balance is the amount of tokens allocated to specific accounts when a blockchain network launches, in this case the gas station account. It allows these accounts to have immediate resources for transactions, testing, or operational roles without needing to acquire tokens afterward.", "", []string{}, []string{}, []string{})

	// Genesis Accounts Tooltips
	GenesisAccountSelectTooltip = ui.NewTooltip("Genesis account", "Genesis accounts are accounts that are created at the genesis block of a blockchain network. They are used to grant initial balances to specific accounts at network launch, enabling early operations without later funding.", "", []string{}, []string{}, []string{})
	GenesisBalanceInputTooltip  = ui.NewTooltip("Genesis balance", "The amount of tokens allocated to specific accounts when a blockchain network launches, in this case the genesis account. It allows these accounts to have immediate resources for transactions, testing, or operational roles without needing to acquire tokens afterward.", "", []string{}, []string{}, []string{})

	// FeeWhitelistAccoutsInputTooltip
	FeeWhitelistAccountsInputTooltip = ui.NewTooltip("Fee whitelist accounts", "Fee-whitelisted accounts are exempt from paying transaction fees, allowing specific accounts to perform transactions without incurring costs. Note that Rollup Operator, Bridge Executor, and Challenger are automatically included in this list.", "", []string{}, []string{}, []string{})
)
