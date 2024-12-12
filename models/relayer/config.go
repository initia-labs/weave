package relayer

import (
	"os"
	"path/filepath"
	"text/template"

	"github.com/initia-labs/weave/types"
)

// EventSource holds event source details
type EventSource struct {
	Mode       string
	URL        string
	BatchDelay string
}

// PacketFilter holds the packet filter configuration
type PacketFilter struct {
	List [][]string
}

type GasPrice struct {
	Amount string
	Denom  string
}

// Data holds all the dynamic data for the template
type Data struct {
	ID            string
	RPCAddr       string
	GRPCAddr      string
	EventSource   EventSource
	PacketFilter  PacketFilter
	ID2           string
	RPCAddr2      string
	GRPCAddr2     string
	EventSource2  EventSource
	PacketFilter2 PacketFilter
	GasPrice2     GasPrice
}

func transformToPacketFilter(pairs []types.IBCChannelPair, isL1 bool) PacketFilter {
	// Initialize the PacketFilter
	packetFilter := PacketFilter{}

	// Transform each IBCChannelPair into a slice of strings
	for _, pair := range pairs {
		var channel types.Channel
		if isL1 {
			channel = pair.L1
		} else {
			channel = pair.L2
		}
		packetFilter.List = append(packetFilter.List, []string{channel.PortID, channel.ChannelID})
	}

	return packetFilter
}

func createHermesConfig(state State) {
	// Define the template directly in a variable
	const configTemplate = `
# The global section has parameters that apply globally to the relayer operation.
[global]

# Specify the verbosity for the relayer logging output. Default: 'info'
# Valid options are 'error', 'warn', 'info', 'debug', 'trace'.
log_level = 'info'


# Specify the mode to be used by the relayer. [Required]
[mode]

# Specify the client mode.
[mode.clients]

# Whether or not to enable the client workers. [Required]
enabled = true

# Whether or not to enable periodic refresh of clients. [Default: true]
# Note: Even if this is disabled, clients will be refreshed automatically if
#      there is activity on a connection or channel they are involved with.
refresh = true

# Whether or not to enable misbehaviour detection for clients. [Default: false]
misbehaviour = true

# Specify the connections mode.
[mode.connections]

# Whether or not to enable the connection workers for handshake completion. [Required]
enabled = true

# Specify the channels mode.
[mode.channels]

# Whether or not to enable the channel workers for handshake completion. [Required]
enabled = true

# Specify the packets mode.
[mode.packets]

# Whether or not to enable the packet workers. [Required]
enabled = true

# Parametrize the periodic packet clearing feature.
# Interval (in number of blocks) at which pending packets
# should be eagerly cleared. A value of '0' will disable
# periodic packet clearing. [Default: 100]
clear_interval = 10

# Whether or not to clear packets on start. [Default: false]
clear_on_start = true

tx_confirmation = true

# The REST section defines parameters for Hermes' built-in RESTful API.
# https://hermes.informal.systems/rest.html
[rest]

# Whether or not to enable the REST service. Default: false
enabled = true

# Specify the IPv4/6 host over which the built-in HTTP server will serve the RESTful
# API requests. Default: 127.0.0.1
host = '127.0.0.1'

# Specify the port over which the built-in HTTP server will serve the restful API
# requests. Default: 3000
port = 7010


# The telemetry section defines parameters for Hermes' built-in telemetry capabilities.
# https://hermes.informal.systems/telemetry.html
[telemetry]

# Whether or not to enable the telemetry service. Default: false
enabled = true

# Specify the IPv4/6 host over which the built-in HTTP server will serve the metrics
# gathered by the telemetry service. Default: 127.0.0.1
host = '127.0.0.1'

# Specify the port over which the built-in HTTP server will serve the metrics gathered
# by the telemetry service. Default: 3001
port = 7011

[[chains]]
id = '{{.ID}}'
type = 'CosmosSdk'
rpc_addr = '{{.RPCAddr}}'
grpc_addr = '{{.GRPCAddr}}'
event_source = { mode = '{{.EventSource.Mode}}', url = '{{.EventSource.URL}}', batch_delay = '{{.EventSource.BatchDelay}}' }
rpc_timeout = '10s'
account_prefix = 'init'
key_name = 'weave-relayer'
store_prefix = 'ibc'
default_gas = 100000
max_gas = 10000000
gas_price = { price = 0.15, denom = 'uinit' }
gas_multiplier = 1.5
max_msg_num = 30
max_tx_size = 2097152
clock_drift = '5s'
max_block_time = '590s'

[chains.packet_filter]
policy = 'allow'
list = [
{{range .PacketFilter.List}}  ['{{index . 0}}', '{{index . 1}}'],
{{end}}
]

[[chains]]
id = '{{.ID2}}'
type = 'CosmosSdk'
rpc_addr = '{{.RPCAddr2}}'
grpc_addr = '{{.GRPCAddr2}}'
event_source = { mode = '{{.EventSource2.Mode}}', url = '{{.EventSource2.URL}}', batch_delay = '{{.EventSource2.BatchDelay}}' }
rpc_timeout = '10s'
account_prefix = 'init'
key_name = 'weave-relayer'
store_prefix = 'ibc'
default_gas = 100000
max_gas = 10000000
gas_price = { price = 0, denom = '{{.GasPrice2.Denom}}' }
gas_multiplier = 1.5
max_msg_num = 30
max_tx_size = 2097152
clock_drift = '5s'
max_block_time = '120s'

[chains.packet_filter]
policy = 'allow'
list = [
{{range .PacketFilter2.List}}  ['{{index . 0}}', '{{index . 1}}'],
{{end}}
]
`

	// Populate data for placeholders
	data := Data{
		ID: state.Config["l1.chain_id"],
		// TODO: revisit
		RPCAddr:  "https://initia-testnet-rpc.polkachu.com/",
		GRPCAddr: "http://" + state.Config["l1.grpc_address"],
		EventSource: EventSource{
			Mode:       "push",
			URL:        state.Config["l1.websocket"],
			BatchDelay: "500ms",
		},
		PacketFilter: transformToPacketFilter(state.IBCChannels, true),
		ID2:          state.Config["l2.chain_id"],
		RPCAddr2:     state.Config["l2.rpc_address"],
		GRPCAddr2:    state.Config["l2.grpc_address"],
		EventSource2: EventSource{
			Mode:       "push",
			URL:        state.Config["l2.websocket"],
			BatchDelay: "500ms",
		},
		GasPrice2: GasPrice{
			Amount: state.Config["l2.gas_price.price"],
			Denom:  state.Config["l2.gas_price.denom"],
		},
		PacketFilter2: transformToPacketFilter(state.IBCChannels, false),
	}

	// Parse the hardcoded template
	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		panic(err)
	}

	homeDir, _ := os.UserHomeDir()
	outputPath := filepath.Join(homeDir, ".hermes", "config.toml")

	// Ensure the directory exists
	err = os.MkdirAll(filepath.Dir(outputPath), 0755) // Creates ~/.hermes if it doesn't exist
	if err != nil {
		panic(err)
	}

	// Open the file for writing
	outputFile, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	// Execute the template with data
	err = tmpl.Execute(outputFile, data)
	if err != nil {
		panic(err)
	}

}
