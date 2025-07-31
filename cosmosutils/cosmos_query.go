package cosmosutils

import (
	"encoding/json"
	"fmt"

	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/types"
)

const (
	NoBalancesText string = "No Balances"
)

func QueryBankBalances(addresses []string, address string) (*Coins, error) {
	return tryEndpoints(
		addresses,
		fmt.Sprintf("/cosmos/bank/v1beta1/balances/%s", address),
		func(data []byte) (*Coins, error) {
			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}

			rawBalances, ok := result["balances"].([]interface{})
			if !ok {
				return nil, fmt.Errorf("failed to parse balances field")
			}

			balancesJSON, err := json.Marshal(rawBalances)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal balances: %w", err)
			}

			var balances Coins
			err = json.Unmarshal(balancesJSON, &balances)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal balances into Coins: %w", err)
			}

			return &balances, nil
		},
	)
}

// tryEndpoints attempts to query from multiple endpoints, returning the first successful result
func tryEndpoints[T any](
	addresses []string,
	path string,
	parseResponse func([]byte) (T, error),
) (T, error) {
	var result T

	if len(addresses) == 0 {
		return result, fmt.Errorf("no LCD endpoints provided")
	}

	httpClient := client.NewHTTPClient()

	// Try each LCD endpoint until one works
	for _, address := range addresses {
		var response interface{}
		if _, err := httpClient.Get(address, path, nil, &response); err != nil {
			continue // Try next endpoint
		}

		// Parse the response
		responseBytes, err := json.Marshal(response)
		if err != nil {
			continue
		}

		if parsed, err := parseResponse(responseBytes); err == nil {
			return parsed, nil
		}
	}

	return result, fmt.Errorf("failed to query from all LCD endpoints")
}

func QueryChannels(addresses []string) (params types.ChannelsResponse, err error) {
	return tryEndpoints(
		addresses,
		"/ibc/core/channel/v1/channels",
		func(data []byte) (types.ChannelsResponse, error) {
			var response types.ChannelsResponse
			err := json.Unmarshal(data, &response)
			return response, err
		},
	)
}

func QueryOPChildParams(addresses []string) (params OPChildParams, err error) {
	response, err := tryEndpoints(
		addresses,
		"/opinit/opchild/v1/params",
		func(data []byte) (OPChildParamsResponse, error) {
			var response OPChildParamsResponse
			err := json.Unmarshal(data, &response)
			return response, err
		},
	)
	if err != nil {
		return params, err
	}
	return response.Params, nil
}
