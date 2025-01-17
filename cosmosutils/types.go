package cosmosutils

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/initia-labs/weave/styles"
)

type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

func (coin Coin) IsZero() bool {
	normalizedAmount := strings.TrimSpace(coin.Amount)
	if normalizedAmount == "" || normalizedAmount == "0" {
		return true
	}

	amountBigInt := new(big.Int)
	amountBigInt, success := amountBigInt.SetString(normalizedAmount, 10)
	if !success {
		return false
	}

	return amountBigInt.Cmp(big.NewInt(0)) == 0
}

type Coins []Coin

func (cs *Coins) Render(maxWidth int) string {
	if len(*cs) == 0 {
		return styles.CreateFrame(NoBalancesText, maxWidth)
	}

	maxAmountLen := cs.getMaxAmountLength()

	var content strings.Builder
	for _, coin := range *cs {
		line := fmt.Sprintf("%-*s %s", maxAmountLen, coin.Amount, coin.Denom)
		content.WriteString(line + "\n")
	}

	contentStr := strings.TrimSuffix(content.String(), "\n")
	return styles.CreateFrame(contentStr, maxWidth)
}

func (cs *Coins) getMaxAmountLength() int {
	maxLen := 0
	for _, coin := range *cs {
		if len(coin.Amount) > maxLen {
			maxLen = len(coin.Amount)
		}
	}
	return maxLen
}

func (cs *Coins) IsZero() bool {
	for _, coin := range *cs {
		if !coin.IsZero() {
			return false
		}
	}
	return true
}

type NodeInfoResponse struct {
	ApplicationVersion struct {
		Version string `json:"version"`
	} `json:"application_version"`
}

type DecCoin struct {
	Denom  string `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom,omitempty"`
	Amount string `protobuf:"bytes,2,opt,name=amount,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"amount"`
}

// DecCoins defines a slice of coins with decimal values
type DecCoins []DecCoin

// Params defines the set of opchild parameters.
type OPChildParams struct {
	// max_validators is the maximum number of validators.
	MaxValidators uint32 `protobuf:"varint,1,opt,name=max_validators,json=maxValidators,proto3" json:"max_validators,omitempty" yaml:"max_validators"`
	// historical_entries is the number of historical entries to persist.
	HistoricalEntries uint32   `protobuf:"varint,2,opt,name=historical_entries,json=historicalEntries,proto3" json:"historical_entries,omitempty" yaml:"historical_entries"`
	MinGasPrices      DecCoins `protobuf:"bytes,3,rep,name=min_gas_prices,json=minGasPrices,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.DecCoins" json:"min_gas_prices" yaml:"min_gas_price"`
	// the account address of bridge executor who can execute permissioned bridge
	// messages.
	BridgeExecutors []string `protobuf:"bytes,4,rep,name=bridge_executors,json=bridgeExecutors,proto3" json:"bridge_executors,omitempty" yaml:"bridge_executors"`
	// the account address of admin who can execute permissioned cosmos messages.
	Admin string `protobuf:"bytes,5,opt,name=admin,proto3" json:"admin,omitempty" yaml:"admin"`
	// the list of addresses that are allowed to pay zero fee.
	FeeWhitelist []string `protobuf:"bytes,6,rep,name=fee_whitelist,json=feeWhitelist,proto3" json:"fee_whitelist,omitempty" yaml:"fee_whitelist"`
	// Max gas for hook execution of `MsgFinalizeTokenDeposit`
	HookMaxGas string `protobuf:"varint,7,opt,name=hook_max_gas,json=hookMaxGas,proto3" json:"hook_max_gas,omitempty" yaml:"hook_max_gas"`
}

type OPChildParamsResponse struct {
	Params OPChildParams `protobuf:"bytes,1,opt,name=params,proto3" json:"params,omitempty"`
}

// ConvertMicroUnits converts micro denominations to their standard units (e.g., uinit -> INIT)
func (c *Coins) ConvertMicroUnits(denomMap map[string]string) *Coins {
	result := make(Coins, len(*c))
	for i, coin := range *c {
		denom := coin.Denom
		if newDenom, ok := denomMap[denom]; ok {
			val, _ := new(big.Int).SetString(coin.Amount, 10)
			div := new(big.Int).SetInt64(1_000_000)
			result[i] = Coin{
				Amount: new(big.Int).Div(val, div).String(),
				Denom:  newDenom,
			}
		} else {
			result[i] = coin
		}
	}
	return &result
}
