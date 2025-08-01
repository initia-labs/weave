package relayer

import (
	"os"
	"path/filepath"
	"slices"
	"text/template"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/types"
)

type Connection struct {
	ConnectionID string
	Channels     []string
}

// PacketFilter holds the packet filter configuration
type PacketFilter struct {
	Connections []Connection
}

type GasPrice struct {
	Amount string
	Denom  string
}

// Data holds all the dynamic data for the template
type Data struct {
	ID            string `toml:"id"`
	RPCAddr       string
	RESTAddr      string
	GasPrice      GasPrice
	Mnemonic      string
	PacketFilter  PacketFilter `toml:"packet_filter"`
	ID2           string
	RPCAddr2      string
	RESTAddr2     string
	GasPrice2     GasPrice
	Mnemonic2     string
	PacketFilter2 PacketFilter
}

// Config defines a structure for the top-level TOML
type Config struct {
	Chains []Data `toml:"chains"`
}

func transformToPacketFilter(pairs []types.IBCChannelPair, isL1 bool) PacketFilter {
	connectionMap := make(map[string][]string)
	// Transform each IBCChannelPair into a slice of strings
	for _, pair := range pairs {
		var connectionID string
		var channel types.Channel
		if isL1 {
			connectionID = pair.L1ConnectionID
			channel = pair.L1
		} else {
			connectionID = pair.L2ConnectionID
			channel = pair.L2
		}

		if connection, ok := connectionMap[connectionID]; !ok {
			connectionMap[connectionID] = []string{channel.ChannelID}
		} else {
			connectionMap[connectionID] = append(connection, channel.ChannelID)
		}
	}

	// Initialize the PacketFilter
	packetFilter := PacketFilter{}
	for connectionID, channels := range connectionMap {
		slices.Sort(channels)
		packetFilter.Connections = append(packetFilter.Connections, Connection{
			ConnectionID: connectionID,
			Channels:     channels,
		})
	}

	return packetFilter
}

func createRapidRelayerConfig(state State) error {
	// Define the template directly in a variable
	const configTemplate = `
{
  "$schema": "./config.schema.json",
  "port": 7010,
  "metricPort": 7011,
  "logLevel": "info",
  "rpcRequestTimeout": 5000,
  "chains": [
    {
      "bech32Prefix": "init",
      "chainId": "{{.ID}}",
      "gasPrice": "{{.GasPrice.Amount}}{{.GasPrice.Denom}}",
      "restUri": ["{{.RESTAddr}}"],
      "rpcUri": ["{{.RPCAddr}}"],
      "wallets": [
        {
          "key": {
            "type": "mnemonic",
            "privateKey": "{{.Mnemonic}}"
          },
          "maxHandlePacket": 10,
          "packetFilter": {
            "connections": [{{range $i, $conn := .PacketFilter.Connections}}{{if $i}}, {{end}}{
              "connectionId": "{{$conn.ConnectionID}}",
              "channels": [{{range $j, $channel := $conn.Channels}}{{if $j}}, {{end}}"{{$channel}}"{{end}}]
            }{{end}}]
          }
        }
      ]
    },
    {
      "bech32Prefix": "init",
      "chainId": "{{.ID2}}",
      "gasPrice": "{{.GasPrice2.Amount}}{{.GasPrice2.Denom}}",
      "restUri": ["{{.RESTAddr2}}"],
      "rpcUri": ["{{.RPCAddr2}}"],
      "wallets": [
        {
          "key": {
            "type": "mnemonic",
            "privateKey": "{{.Mnemonic2}}"
          },
          "maxHandlePacket": 10,
          "packetFilter": {
            "connections": [{{range $i, $conn := .PacketFilter2.Connections}}{{if $i}}, {{end}}{
              "connectionId": "{{$conn.ConnectionID}}",
              "channels": [{{range $j, $channel := $conn.Channels}}{{if $j}}, {{end}}"{{$channel}}"{{end}}]
            }{{end}}]
          }
        }
      ]
    }
  ]
}
`

	// Populate data for placeholders
	data := Data{
		ID:       state.Config["l1.chain_id"],
		RPCAddr:  state.Config["l1.rpc_address"],
		RESTAddr: state.Config["l1.lcd_address"],
		GasPrice: GasPrice{
			Amount: state.Config["l1.gas_price.price"],
			Denom:  state.Config["l1.gas_price.denom"],
		},
		Mnemonic:     state.l1RelayerMnemonic,
		PacketFilter: transformToPacketFilter(state.IBCChannels, true),

		ID2:       state.Config["l2.chain_id"],
		RPCAddr2:  state.Config["l2.rpc_address"],
		RESTAddr2: state.Config["l2.lcd_address"],
		GasPrice2: GasPrice{
			Amount: state.Config["l2.gas_price.price"],
			Denom:  state.Config["l2.gas_price.denom"],
		},
		Mnemonic2:     state.l2RelayerMnemonic,
		PacketFilter2: transformToPacketFilter(state.IBCChannels, false),
	}

	// Parse the hardcoded template
	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return err
	}

	homeDir, _ := os.UserHomeDir()
	outputPath := filepath.Join(homeDir, common.RelayerConfigPath)

	// Ensure the directory exists
	err = os.MkdirAll(filepath.Dir(outputPath), 0o755) // Creates ~/.relayer if it doesn't exist
	if err != nil {
		return err
	}

	// Open the file for writing
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Execute the template with data
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		return err
	}

	return nil
}
