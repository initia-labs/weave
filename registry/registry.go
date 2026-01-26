package registry

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/types"
)

// TODO: HOTFIX for un-stable RPCs and LCDs
var (
	CELESTIA_MAINNET_RPCS = []Endpoint{
		{
			Address: "https://rpc.lunaroasis.net",
		},
		{
			Address: "https://rpc.celestia.nodestake.org",
		},
		{
			Address: "https://rpc.lavenderfive.com:443/celestia",
		},
	}
	CELESTIA_TESTNET_RPCS = []Endpoint{
		{
			Address: "https://rpc-mocha.pops.one",
		},
		{
			Address: "https://celestia-testnet-rpc.itrocket.net",
		},
		{
			Address: "https://rpc.celestia.testnet.dteam.tech:443",
		},
	}
	INITIA_MAINNET_RPCS = []Endpoint{
		{
			Address: "https://rpc.initia.xyz",
		},
		{
			Address: "https://initia-rpc.cosmosspaces.zone",
		},
		{
			Address: "https://initia-archive.cosmosspaces.zone",
		},
	}
	INITIA_TESTNET_RPCS = []Endpoint{
		{
			Address: "https://rpc.testnet.initia.xyz",
		},
	}

	CELESTIA_MAINNET_LCDS = []Endpoint{
		{
			Address: "https://api.lunaroasis.net",
		},
		{
			Address: "https://api.celestia.nodestake.org",
		},
		{
			Address: "https://rest.lavenderfive.com:443/celestia",
		},
	}
	CELESTIA_TESTNET_LCDS = []Endpoint{
		{
			Address: "https://celestia-testnet-api.polkachu.com",
		},
		{
			Address: "https://celestia-testnet-api.itrocket.net",
		},
		{
			Address: "https://api.celestia.testnet.dteam.tech",
		},
	}
	INITIA_MAINNET_LCDS = []Endpoint{
		{
			Address: "https://rest.initia.xyz",
		},
		{
			Address: "https://initia-api.polkachu.com",
		},
	}
	INITIA_TESTNET_LCDS = []Endpoint{
		{
			Address: "https://rest.testnet.initia.xyz",
		},
	}
)

// LoadedChainRegistry contains a map of chain id to the chain.json
var LoadedChainRegistry = make(map[ChainType]*ChainRegistry)

const (
	MAX_FALLBACK_RPCS = 3
	MAX_FALLBACK_LCDS = 3
)

type ChainRegistry struct {
	ChainId      string   `json:"chain_id"`
	PrettyName   string   `json:"pretty_name"`
	Bech32Prefix string   `json:"bech32_prefix"`
	Fees         Fees     `json:"fees"`
	Codebase     Codebase `json:"codebase"`
	Apis         Apis     `json:"apis"`
	Peers        Peers    `json:"peers"`

	ActiveRpcs []string
	ActiveLcds []string
}

type Fees struct {
	FeeTokens []FeeTokens `json:"fee_tokens"`
}

type FeeTokens struct {
	Denom            string  `json:"denom"`
	FixedMinGasPrice float64 `json:"fixed_min_gas_price"`
}

type Codebase struct {
	Genesis Genesis `json:"genesis"`
}

type Genesis struct {
	GenesisUrl string `json:"genesis_url"`
}

type Apis struct {
	Rpc  []Endpoint `json:"rpc"`
	Rest []Endpoint `json:"rest"`
	Grpc []Endpoint `json:"grpc"`
}

type Endpoint struct {
	Address        string `json:"address"`
	Provider       string `json:"provider"`
	AuthorizedUser string `json:"authorizedUser,omitempty"`
	IndexForSkip   int    `json:"indexForSkip,omitempty"`
}

type Peers struct {
	Seeds           []Peer `json:"seeds,omitempty"`
	PersistentPeers []Peer `json:"persistent_peers,omitempty"`
}

type Peer struct {
	Id       string `json:"id"`
	Address  string `json:"address"`
	Provider string `json:"provider,omitempty"`
}

type Channel struct {
	Channel struct {
		ConnectionHops []string `json:"connection_hops"`
		Counterparty   struct {
			ChannelID string `json:"channel_id"`
			PortID    string `json:"port_id"`
		} `json:"counterparty"`
	} `json:"channel"`
}

type Connection struct {
	Connection struct {
		Counterparty struct {
			ClientID string `json:"client_id"`
		} `json:"counterparty"`
	} `json:"connection"`
}

func (cr *ChainRegistry) GetChainId() string {
	return cr.ChainId
}

func (cr *ChainRegistry) GetBech32Prefix() string {
	return cr.Bech32Prefix
}

func (cr *ChainRegistry) GetMinGasPriceByDenom(denom string) (string, error) {
	for _, feeToken := range cr.Fees.FeeTokens {
		if feeToken.Denom == denom {
			return fmt.Sprintf("%g%s", feeToken.FixedMinGasPrice, denom), nil
		}
	}
	return "", fmt.Errorf("denomination %s not found in fee tokens", denom)
}

func (cr *ChainRegistry) GetFixedMinGasPriceByDenom(denom string) (string, error) {
	for _, feeToken := range cr.Fees.FeeTokens {
		if feeToken.Denom == denom {
			return fmt.Sprintf("%g", feeToken.FixedMinGasPrice), nil
		}
	}
	return "", fmt.Errorf("denomination %s not found in fee tokens", denom)
}

func checkAndAddPort(addr string) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return "", fmt.Errorf("invalid address: %v", err)
	}

	if u.Port() == "" {
		if u.Scheme == "https" {
			u.Host = u.Host + ":443"
		} else if u.Scheme == "http" {
			u.Host = u.Host + ":80"
		}
	}

	return u.String(), nil
}

func (cr *ChainRegistry) GetActiveRpcs() ([]string, error) {
	if len(cr.ActiveRpcs) > 0 {
		return cr.ActiveRpcs, nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var activeRpcs []string
	var done bool // Flag to signal early termination

	// Create a channel to limit concurrent requests
	semaphore := make(chan struct{}, MAX_FALLBACK_RPCS)

	for _, rpc := range cr.Apis.Rpc {
		if rpc.AuthorizedUser != "" {
			continue
		}

		wg.Add(1)
		go func(endpoint Endpoint) {
			defer wg.Done()

			// Check if we're already done
			mu.Lock()
			if done {
				mu.Unlock()
				return
			}
			mu.Unlock()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			address, err := checkAndAddPort(endpoint.Address)
			if err != nil {
				return
			}

			httpClient := client.NewHTTPClient()
			_, err = httpClient.Get(address, "/health", nil, nil)
			if err != nil {
				return
			}

			// Thread-safe append and check for early termination
			mu.Lock()
			defer mu.Unlock()

			if done {
				return
			}

			activeRpcs = append(activeRpcs, address)
			if len(activeRpcs) >= MAX_FALLBACK_RPCS {
				done = true
			}
		}(rpc)
	}

	wg.Wait()

	if len(activeRpcs) == 0 {
		return nil, fmt.Errorf("no active RPC endpoints available")
	}

	cr.ActiveRpcs = activeRpcs
	return cr.ActiveRpcs, nil
}

func (cr *ChainRegistry) GetFirstActiveRpc() (string, error) {
	rpcs, err := cr.GetActiveRpcs()
	if err != nil {
		return "", err
	}
	if len(rpcs) == 0 {
		return "", fmt.Errorf("no active RPC endpoints available")
	}
	return rpcs[0], nil
}

func (cr *ChainRegistry) GetActiveLcds() ([]string, error) {
	if len(cr.ActiveLcds) > 0 {
		return cr.ActiveLcds, nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var activeLcds []string
	var done bool // Flag to signal early termination

	// Create a channel to limit concurrent requests
	semaphore := make(chan struct{}, MAX_FALLBACK_LCDS)

	for _, lcd := range cr.Apis.Rest {
		if lcd.AuthorizedUser != "" {
			continue
		}

		wg.Add(1)
		go func(endpoint Endpoint) {
			defer wg.Done()

			// Check if we're already done
			mu.Lock()
			if done {
				mu.Unlock()
				return
			}
			mu.Unlock()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			address, err := checkAndAddPort(endpoint.Address)
			if err != nil {
				return
			}

			httpClient := client.NewHTTPClient()
			_, err = httpClient.Get(address, "/cosmos/base/tendermint/v1beta1/syncing", nil, nil)
			if err != nil {
				return
			}

			// Thread-safe append and check for early termination
			mu.Lock()
			defer mu.Unlock()

			if done {
				return
			}

			activeLcds = append(activeLcds, address)
			if len(activeLcds) >= MAX_FALLBACK_LCDS {
				done = true
			}
		}(lcd)
	}

	wg.Wait()

	if len(activeLcds) == 0 {
		return nil, fmt.Errorf("no active LCD endpoints available")
	}

	cr.ActiveLcds = activeLcds
	return cr.ActiveLcds, nil
}

func (cr *ChainRegistry) GetFirstActiveLcd() (string, error) {
	lcds, err := cr.GetActiveLcds()
	if err != nil {
		return "", err
	}
	if len(lcds) == 0 {
		return "", fmt.Errorf("no active LCD endpoints available")
	}
	return lcds[0], nil
}

// queryActiveEndpoints is a helper function that queries active LCD endpoints
// and returns the first successful response
func (cr *ChainRegistry) queryActiveEndpoints(path string, result interface{}) error {
	addresses, err := cr.GetActiveLcds()
	if err != nil {
		return err
	}
	httpClient := client.NewHTTPClient()

	for _, address := range addresses {
		if _, err := httpClient.Get(address, path, nil, result); err != nil {
			continue
		}
		return nil
	}

	return fmt.Errorf("failed to query %s from any active LCD endpoint", path)
}

func (cr *ChainRegistry) GetOpinitBridgeInfo(id string) (types.Bridge, error) {
	var bridgeInfo types.Bridge
	path := fmt.Sprintf("/opinit/ophost/v1/bridges/%s", id)

	if err := cr.queryActiveEndpoints(path, &bridgeInfo); err != nil {
		return types.Bridge{}, fmt.Errorf("failed to get opinit bridge info: %w", err)
	}

	return bridgeInfo, nil
}

func (cr *ChainRegistry) GetIBCChannelInfo(port, channel string) (types.ChannelResponse, error) {
	var response types.ChannelResponse
	path := fmt.Sprintf("/ibc/core/channel/v1/channels/%s/ports/%s", channel, port)

	if err := cr.queryActiveEndpoints(path, &response); err != nil {
		return types.ChannelResponse{}, fmt.Errorf("failed to get counterparty IBC channel: %w", err)
	}

	if len(response.Channel.ConnectionHops) == 0 {
		return types.ChannelResponse{}, fmt.Errorf("no connection ID found")
	}

	return response, nil
}

func normalizeGRPCAddress(addr string) (string, error) {
	if strings.Contains(addr, "://") {
		if !strings.HasPrefix(addr, "https://") {
			return "", fmt.Errorf("only https:// protocol is allowed")
		}
		return addr, nil
	}
	addr = "https://" + addr
	return addr, nil
}

func (cr *ChainRegistry) GetActiveGrpc() (string, error) {
	grpcClient := client.NewGRPCClient()
	for _, grpc := range cr.Apis.Grpc {
		err := grpcClient.CheckHealth(grpc.Address)
		if err != nil {
			continue
		}

		addr, err := normalizeGRPCAddress(grpc.Address)
		if err != nil {
			continue
		}
		return addr, nil
	}

	return "", fmt.Errorf("no active gRPC endpoints available")
}

// normalizeRPCToWebSocket converts an RPC endpoint (HTTP/HTTPS) to WebSocket (WS/WSS).
func normalizeRPCToWebSocket(rpcEndpoint string) (string, error) {
	// Parse the URL
	u, err := url.Parse(rpcEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid RPC endpoint: %v", err)
	}

	// Convert HTTP(S) to WS(S)
	switch u.Scheme {
	case "http":
		u.Scheme = "ws" // Convert HTTP to WS
	case "https":
		u.Scheme = "wss" // Convert HTTPS to WSS
	default:
		return "", fmt.Errorf("unsupported scheme for RPC to WebSocket conversion: %s", u.Scheme)
	}

	return u.String() + "/websocket", nil
}

func (cr *ChainRegistry) GetActiveWebSocket() (string, error) {
	rpc, err := cr.GetActiveRpcs()
	if err != nil {
		return "", fmt.Errorf("failed to get RPC endpoint: %v", err)
	}
	for _, rpc := range rpc {
		websocket, err := normalizeRPCToWebSocket(rpc)
		if err != nil {
			continue
		}
		return websocket, nil
	}
	return "", fmt.Errorf("no active WebSocket endpoints available")
}

func (cr *ChainRegistry) GetSeeds() string {
	var seeds []string
	for _, seed := range cr.Peers.Seeds {
		seeds = append(seeds, fmt.Sprintf("%s@%s", seed.Id, seed.Address))
	}
	return strings.Join(seeds, ",")
}

func (cr *ChainRegistry) GetPersistentPeers() string {
	var persistentPeers []string
	for _, seed := range cr.Peers.PersistentPeers {
		persistentPeers = append(persistentPeers, fmt.Sprintf("%s@%s", seed.Id, seed.Address))
	}
	return strings.Join(persistentPeers, ",")
}

func (cr *ChainRegistry) GetGenesisUrl() string {
	return cr.Codebase.Genesis.GenesisUrl
}

func (cr *ChainRegistry) GetDefaultFeeToken() (FeeTokens, error) {
	for _, feeToken := range cr.Fees.FeeTokens {
		return feeToken, nil
	}
	return FeeTokens{}, fmt.Errorf("fee token not found")
}

func (cr *ChainRegistry) GetDefaultMinGasPrices() (string, error) {
	feeToken, err := cr.GetDefaultFeeToken()
	if err != nil {
		return "", fmt.Errorf("failed to get default fee token: %v", err)
	}

	return fmt.Sprintf("%s%s", strconv.FormatFloat(feeToken.FixedMinGasPrice, 'f', -1, 64), feeToken.Denom), nil
}

func loadChainRegistry(chainType ChainType) error {
	httpClient := client.NewHTTPClient()
	endpoint := GetRegistryEndpoint(chainType)
	LoadedChainRegistry[chainType] = &ChainRegistry{}
	if _, err := httpClient.Get(endpoint, "", nil, LoadedChainRegistry[chainType]); err != nil {
		return err
	}
	if err := replaceRpcsAndLcds(chainType, LoadedChainRegistry[chainType]); err != nil {
		return fmt.Errorf("failed to replace RPCs and LCDs for %s: %v", chainType, err)
	}

	return nil
}

func replaceRpcsAndLcds(chainType ChainType, chainRegistry *ChainRegistry) error {
	switch chainType {
	case CelestiaMainnet:
		chainRegistry.Apis.Rpc = CELESTIA_MAINNET_RPCS
		chainRegistry.Apis.Rest = CELESTIA_MAINNET_LCDS
	case CelestiaTestnet:
		chainRegistry.Apis.Rpc = CELESTIA_TESTNET_RPCS
		chainRegistry.Apis.Rest = CELESTIA_TESTNET_LCDS
	case InitiaL1Mainnet:
		chainRegistry.Apis.Rpc = INITIA_MAINNET_RPCS
		chainRegistry.Apis.Rest = INITIA_MAINNET_LCDS
	case InitiaL1Testnet:
		chainRegistry.Apis.Rpc = INITIA_TESTNET_RPCS
		chainRegistry.Apis.Rest = INITIA_TESTNET_LCDS
	}

	return nil
}

func GetChainRegistry(chainType ChainType) (*ChainRegistry, error) {
	chainRegistry, ok := LoadedChainRegistry[chainType]
	if !ok {
		if err := loadChainRegistry(chainType); err != nil {
			return nil, fmt.Errorf("failed to load chain registry for %s: %v", chainType, err)
		}
		return LoadedChainRegistry[chainType], nil
	}

	return chainRegistry, nil
}

type ChainRegistryWithChainType struct {
	ChainRegistry
	ChainType ChainType
}

// LoadedL2Registry contains a map of l2 chain id to the chain.json with [testnet|mainnet] specified
var LoadedL2Registry = make(map[string]*ChainRegistryWithChainType)

func loadL2RegistryForType(chainType ChainType) error {
	httpClient := client.NewHTTPClient()

	var chains []*ChainRegistry
	apiURL := ChainTypeToInitiaRegistryAPI[chainType]
	if _, err := httpClient.Get(apiURL, "", nil, &chains); err != nil {
		return fmt.Errorf("failed to fetch registry from %s: %w", apiURL, err)
	}

	for _, chain := range chains {
		if chain.PrettyName == InitiaL1PrettyName {
			continue
		}
		LoadedL2Registry[chain.GetChainId()] = &ChainRegistryWithChainType{
			ChainRegistry: *chain,
			ChainType:     chainType,
		}
	}
	return nil
}

func GetL2Registry(chainType ChainType, chainId string) (*ChainRegistry, error) {
	if registry, ok := LoadedL2Registry[chainId]; ok {
		return &registry.ChainRegistry, nil
	}

	err := loadL2RegistryForType(chainType)
	if err != nil {
		return nil, fmt.Errorf("failed to load L2 registry: %w", err)
	}

	registry, ok := LoadedL2Registry[chainId]
	if !ok {
		return nil, fmt.Errorf("chain id %s not found in remote registry", chainId)
	}

	return &registry.ChainRegistry, nil
}

type L2AvailableNetwork struct {
	PrettyName string
	ChainId    string
}

func GetAllL2AvailableNetwork(chainType ChainType) ([]L2AvailableNetwork, error) {
	if len(LoadedL2Registry) == 0 {
		if err := loadL2RegistryForType(chainType); err != nil {
			return nil, fmt.Errorf("failed to load L2 registry: %v", err)
		}
	}

	var networks []L2AvailableNetwork

	for _, registry := range LoadedL2Registry {
		if registry.ChainType == chainType {
			networks = append(networks, L2AvailableNetwork{
				PrettyName: registry.PrettyName,
				ChainId:    registry.ChainId,
			})
		}
	}

	return networks, nil
}

var OPInitBotsSpecVersion map[string]int

func loadOPInitBotsSpecVersion() error {
	httpClient := client.NewHTTPClient()
	if _, err := httpClient.Get(OPInitBotsSpecEndpoint, "", nil, &OPInitBotsSpecVersion); err != nil {
		return fmt.Errorf("failed to load opinit spec_version: %v", err)
	}
	return nil
}

func GetOPInitBotsSpecVersion(chainId string) (int, error) {
	if OPInitBotsSpecVersion == nil {
		if err := loadOPInitBotsSpecVersion(); err != nil {
			return 0, err
		}
	}

	version, ok := OPInitBotsSpecVersion[chainId]
	if !ok {
		return 0, fmt.Errorf("chain id not found in the spec_version")
	}

	return version, nil
}

func (cr *ChainRegistry) GetCounterpartyClientId(portID, channelID string) (Connection, error) {
	// First, get the channel information
	var channel Channel
	channelPath := fmt.Sprintf("/ibc/core/channel/v1/channels/%s/ports/%s", channelID, portID)
	if err := cr.queryActiveEndpoints(channelPath, &channel); err != nil {
		return Connection{}, fmt.Errorf("failed to get channel info: %w", err)
	}

	// Validate connection hops
	if len(channel.Channel.ConnectionHops) == 0 {
		return Connection{}, fmt.Errorf("no connection hops found for channel %s", channelID)
	}

	// Get the connection information using the first connection hop
	var connection Connection
	connectionPath := fmt.Sprintf("/ibc/core/connection/v1/connections/%s", channel.Channel.ConnectionHops[0])
	if err := cr.queryActiveEndpoints(connectionPath, &connection); err != nil {
		return Connection{}, fmt.Errorf("failed to get connection info: %w", err)
	}

	return connection, nil
}

func GetInitiaGraphQLFromType(chainType ChainType) (string, error) {
	apiURL, ok := ChainTypeToInitiaGraphQLAPI[chainType]
	if ok {
		return apiURL, nil
	}
	return "", fmt.Errorf("graphql for chain type %s not found", chainType)
}
