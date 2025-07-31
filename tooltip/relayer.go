package tooltip

import (
	"github.com/initia-labs/weave/ui"
)

var (
	// Rollup select
	RelayerRollupSelectLocalTooltip       = ui.NewTooltip("Local rollup", "Run a relayer for the rollup that you just launched locally. Using artifacts from .minitia/artifacts to setup the relayer.", "", []string{}, []string{}, []string{})
	RelayerRollupSelectWhitelistedTooltip = ui.NewTooltip("Whitelisted rollup", "Run a relayer for any live rollup in https://github.com/initia-labs/initia-registry.", "", []string{}, []string{}, []string{})
	RelayerRollupSelectManualTooltip      = ui.NewTooltip("Manual Relayer Setup", "Setup the relayer manually by providing the L1 and rollup chain IDs, RPC endpoints, GRPC endpoints, and more.", "", []string{}, []string{}, []string{})

	// L1 network select
	RelayerL1NetworkSelectTooltip = ui.NewTooltip("L1 Network to relay messages", "Testnet (initiation-2) is the only supported network for now.", "", []string{}, []string{}, []string{})

	// Rollup LCD endpoint
	RelayerRollupLCDTooltip = ui.NewTooltip("Rollup LCD endpoint", "LCD endpoint to your rollup node server. By providing this, relayer will be able to fetch the IBC channels and ports from the rollup node server.", "", []string{}, []string{}, []string{})

	// IBC channels setup
	RelayerIBCMinimalSetupTooltip    = ui.NewTooltip("Minimal setup", "Subscribe to only `transfer` and `nft-transfer` IBC Channels created when launching the rollup with `minitiad launch` or `weave rollup launch`. This is recommended for new networks or local testing.", "", []string{}, []string{}, []string{})
	RelayerIBCFillFromLCDTooltip     = ui.NewTooltip("Get all available IBC Channels", "By filling in the rollup LCD endpoint, Weave will be able to detect all available IBC Channels and show you all the IBC Channel pairs.", "", []string{}, []string{}, []string{})
	RelayerIBCManualSetupTooltip     = ui.NewTooltip("Manual setup", "Setup each IBC Channel manually by specifying the port ID and channel ID.", "", []string{}, []string{}, []string{})
	RelayerIBCChannelsTooltip        = ui.NewTooltip("IBC Channels", "Relayer will listen to the selected channels (and ports) and relay messages between L1 and rollup network. Relay all option is recommended if you don't want to miss any messages.\n\nRefer to https://ibc.cosmos.network/main/ibc/overview.html#channels for more information.", "", []string{}, []string{}, []string{})
	RelayerL1IBCPortIDTooltip        = ui.NewTooltip("Port ID on L1", "Port Identifier for the IBC channel on L1 that the relayer should relay messages. Refer to https://ibc.cosmos.network/main/ibc/overview/#ports for more information.", "", []string{}, []string{}, []string{})
	RelayerL1IBCChannelIDTooltip     = ui.NewTooltip("Channel ID on L1", "Channel Identifier for the IBC channel on L1 that the relayer should relay messages. Refer to https://ibc.cosmos.network/main/ibc/overview/#channels for more information.", "", []string{}, []string{}, []string{})
	RelayerRollupIBCPortIDTooltip    = ui.NewTooltip("Port ID on rollup", "Port Identifier for the IBC channel on rollup that the relayer should relay messages. Refer to https://ibc.cosmos.network/main/ibc/overview/#ports for more information.", "", []string{}, []string{}, []string{})
	RelayerRollupIBCChannelIDTooltip = ui.NewTooltip("Channel ID on rollup", "Channel Identifier for the IBC channel on rollup that the relayer should relay messages. Refer to https://ibc.cosmos.network/main/ibc/overview/#channels for more information.", "", []string{}, []string{}, []string{})

	// Relayer key
	RelayerChallengerKeyTooltip   = ui.NewTooltip("Relayer with Challenger Key", "This is recommended because challenger account is exempted from gas fees on the rollup and able to stop other relayers from relaying when it detects a malicious message coming from it. If you want to setup relayer with a separate key, select No.", "", []string{}, []string{}, []string{})
	RelayerL1KeySelectTooltip     = ui.NewTooltip("L1 Relayer Key", "The key/account that the relayer will use to interact with the L1 network.", "", []string{}, []string{}, []string{})
	RelayerRollupKeySelectTooltip = ui.NewTooltip("Rollup Relayer Key", "The key/account that the relayer will use to interact with the rollup network.", "", []string{}, []string{}, []string{})

	// Funding amount select
	RelayerFundingAmountSelectTooltip = ui.NewTooltip("Funding the relayer accounts", "Relayer accounts on both the Initia L1 and the rollup requires gas token funding to ensure the relayer can function properly.", "", []string{}, []string{}, []string{})
)
