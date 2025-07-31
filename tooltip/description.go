package tooltip

import "fmt"

func ChainIDDescription(networkType string) string {
	return fmt.Sprintf("Identifier for the %s network.", networkType)
}

func RPCEndpointDescription(networkType string) string {
	return fmt.Sprintf("The network address and port that the %s RPC node will listen to. This endpoint is used by the rollup bots to communicate with the %s network and for users to submit transactions.", networkType, networkType)
}

func RESTEndpointDescription(networkType string) string {
	return fmt.Sprintf("The network address and port that an %s REST node will listen to. This allows the rollup bots to query additional data from the %s network.", networkType, networkType)
}

func GasDenomDescription(networkType string) string {
	return fmt.Sprintf("The gas token denom to be used for submitting transactions to the %s node.", networkType)
}

func GasPriceDescription(networkType string) string {
	return fmt.Sprintf("The gas price to be used for submitting transactions to the %s node. This value should be set to the minimum gas price for the %s node.", networkType, networkType)
}

func MinGasPriceDescription(networkType string) string {
	return fmt.Sprintf("The minimum gas price for transactions submitted on the %s network. Any transactions submitted to the %s network with a lower gas price will be rejected.", networkType, networkType)
}
