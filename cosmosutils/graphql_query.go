package cosmosutils

import (
	"fmt"

	"github.com/initia-labs/weave/client"
)

type TransactionEventsResponse struct {
	Data struct {
		TransactionEvents []struct {
			BlockHeight     int    `json:"block_height,omitempty"`
			TransactionHash string `json:"transaction_hash,omitempty"`
		} `json:"transaction_events"`
	} `json:"data"`
}

func QueryCreateBridgeHeight(client *client.GraphQLClient, bridgeId string) (int, error) {
	query := `
		query CreateBridgeHeight($bridgeId: String) {
			transaction_events(
			where: {event_key: {_eq: "create_bridge.bridge_id"}, event_value: {_eq: $bridgeId}}
			limit: 1
			order_by: {block_height: desc}
		  ) {
			block_height
		  }
		}
	`
	var response TransactionEventsResponse
	err := client.Query(
		query,
		map[string]interface{}{
			"bridgeId": bridgeId,
		},
		&response,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to query CreateBridgeHeight: %w", err)
	}
	if len(response.Data.TransactionEvents) != 1 {
		return 0, fmt.Errorf("expected 1 CreateBridgeHeight event, got %d", len(response.Data.TransactionEvents))
	}

	return response.Data.TransactionEvents[0].BlockHeight, err
}

func QueryLatestDepositHeight(client *client.GraphQLClient, bridgeId, sequence string) (int, error) {
	txSequenceQuery := `
		query TransactionWithL1Sequence($sequence: String, $offset: Int) {
			transaction_events(
				where: {event_key: {_eq: "initiate_token_deposit.l1_sequence"}, event_value: {_eq: $sequence}}
				limit: 100
				offset: $offset
				order_by: {block_height: desc}
			) {
				transaction_hash
				block_height
			}
		}
	`
	txBridgeQuery := `
		query HeightWithBridgeTransaction($bridgeId: String, $txHash: String, $height: bigint) {
			transaction_events(
				where: {transaction_hash: {_eq: $txHash}, block_height: {_eq: $height}, event_key: {_eq: "initiate_token_deposit.bridge_id"}, event_value: {_eq: $bridgeId}}
				limit: 1
				order_by: {block_height: desc}
			) {
				block_height
			}
		}
	`

	pageSize := 100
	for offset := 0; ; offset += pageSize {
		var seqResponse TransactionEventsResponse
		err := client.Query(
			txSequenceQuery,
			map[string]interface{}{
				"sequence": sequence,
				"offset":   offset,
			},
			&seqResponse,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to query LatestDepositHeight: %w", err)
		}
		if len(seqResponse.Data.TransactionEvents) == 0 {
			break
		}

		for _, transactionEvent := range seqResponse.Data.TransactionEvents {
			var bridgeResponse TransactionEventsResponse
			err = client.Query(
				txBridgeQuery,
				map[string]interface{}{
					"bridgeId": bridgeId,
					"txHash":   transactionEvent.TransactionHash,
					"height":   transactionEvent.BlockHeight,
				},
				&bridgeResponse,
			)
			if err != nil {
				return 0, fmt.Errorf("failed to query HeightWithBridgeTransaction: %w", err)
			}
			if len(bridgeResponse.Data.TransactionEvents) == 1 {
				return bridgeResponse.Data.TransactionEvents[0].BlockHeight, nil
			}
		}
	}
	return 0, fmt.Errorf("cannot find deposit height according to the given bridge and sequence")
}
