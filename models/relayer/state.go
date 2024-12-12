package relayer

import (
	"github.com/initia-labs/weave/types"
)

type State struct {
	weave       types.WeaveState
	Config      map[string]string
	IBCChannels []types.IBCChannelPair

	l1KeyMethod       string
	l1RelayerAddress  string
	l1RelayerMnemonic string
	l1NeedsFunding    bool
	l1FundingAmount   string
	l1FundingTxHash   string

	l2KeyMethod       string
	l2RelayerAddress  string
	l2RelayerMnemonic string
	l2NeedsFunding    bool
	l2FundingAmount   string
	l2FundingTxHash   string

	hermesBinaryPath string
}

func NewRelayerState() State {
	return State{
		weave:       types.NewWeaveState(),
		Config:      make(map[string]string),
		IBCChannels: make([]types.IBCChannelPair, 0),
	}
}

func (state State) Clone() State {
	config := make(map[string]string)
	for k, v := range state.Config {
		config[k] = v
	}
	clone := State{
		weave:       state.weave,
		Config:      config,
		IBCChannels: state.IBCChannels,

		l1KeyMethod:       state.l1KeyMethod,
		l1RelayerAddress:  state.l1RelayerAddress,
		l1RelayerMnemonic: state.l1RelayerMnemonic,
		l1NeedsFunding:    state.l1NeedsFunding,
		l1FundingAmount:   state.l1FundingAmount,
		l1FundingTxHash:   state.l1FundingTxHash,

		l2KeyMethod:       state.l2KeyMethod,
		l2RelayerAddress:  state.l2RelayerAddress,
		l2RelayerMnemonic: state.l2RelayerMnemonic,
		l2NeedsFunding:    state.l2NeedsFunding,
		l2FundingAmount:   state.l2FundingAmount,
		l2FundingTxHash:   state.l2FundingTxHash,

		hermesBinaryPath: state.hermesBinaryPath,
	}

	return clone
}
