package registry

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/initia-labs/weave/client"
	"github.com/initia-labs/weave/types"
)

// LoadedChainRegistry contains a map of chain id to the chain.json
var LoadedChainRegistry = make(map[ChainType]*ChainRegistry)

type ChainRegistry struct {
	ChainId      string   `json:"chain_id"`
	PrettyName   string   `json:"pretty_name"`
	Bech32Prefix string   `json:"bech32_prefix"`
	Fees         Fees     `json:"fees"`
	Codebase     Codebase `json:"codebase"`
	Apis         Apis     `json:"apis"`
	Peers        Peers    `json:"peers"`
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

func (cr *ChainRegistry) MustGetFixedMinGasPriceByDenom(denom string) string {
	for _, feeToken := range cr.Fees.FeeTokens {
		if feeToken.Denom == denom {
			return fmt.Sprintf("%g", feeToken.FixedMinGasPrice)
		}
	}
	panic(fmt.Errorf("denomination %s not found in fee tokens", denom))
}

func (cr *ChainRegistry) MustGetMinGasPriceByDenom(denom string) string {
	minGasPrice, err := cr.GetMinGasPriceByDenom(denom)
	if err != nil {
		panic(err)
	}

	return minGasPrice
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

func (cr *ChainRegistry) GetActiveRpc() (string, error) {
	httpClient := client.NewHTTPClient()
	for _, rpc := range cr.Apis.Rpc {
		address, err := checkAndAddPort(rpc.Address)
		if err != nil {
			continue
		}

		_, err = httpClient.Get(address, "/health", nil, nil)
		if err != nil {
			continue
		}

		return address, nil
	}

	return "", fmt.Errorf("no active RPC endpoints available")
}

func (cr *ChainRegistry) MustGetActiveRpc() string {
	rpc, err := cr.GetActiveRpc()
	if err != nil {
		panic(err)
	}

	return rpc
}

func (cr *ChainRegistry) GetActiveLcd() (string, error) {
	httpClient := client.NewHTTPClient()
	for _, lcd := range cr.Apis.Rest {
		_, err := httpClient.Get(lcd.Address, "/cosmos/base/tendermint/v1beta1/syncing", nil, nil)
		if err != nil {
			continue
		}

		return lcd.Address, nil
	}

	return "", fmt.Errorf("no active LCD endpoints available")
}

func (cr *ChainRegistry) MustGetActiveLcd() string {
	lcd, err := cr.GetActiveLcd()
	if err != nil {
		panic(err)
	}

	return lcd
}

func (cr *ChainRegistry) MustGetOpinitBridgeInfo(id string) types.Bridge {
	address := cr.MustGetActiveLcd()
	httpClient := client.NewHTTPClient()

	var bridgeInfo types.Bridge
	if _, err := httpClient.Get(address, fmt.Sprintf("/opinit/ophost/v1/bridges/%s", id), nil, &bridgeInfo); err != nil {
		panic(err)
	}
	return bridgeInfo
}

func (cr *ChainRegistry) MustGetCounterPartyIBCChannel(port, channel string) types.Channel {
	address := cr.MustGetActiveLcd()
	httpClient := client.NewHTTPClient()

	var response types.MinimalIBCChannelResponse
	if _, err := httpClient.Get(address, fmt.Sprintf("/ibc/core/channel/v1/channels/%s/ports/%s", channel, port), nil, &response); err != nil {
		panic(err)
	}
	return response.Channel.Counterparty
}

func normalizeGRPCAddress(addr string) (string, error) {
	// Parse the URL
	u, err := url.Parse(addr)
	if err != nil {
		return "", fmt.Errorf("invalid address: %v", err)
	}

	if u.Scheme == "grpc" {
		u.Scheme = "http"
	}

	// Add default port if missing
	if u.Port() == "" {
		u.Host = u.Host + ":9090"
	}

	return u.String(), nil
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

func (cr *ChainRegistry) MustGetActiveGrpc() string {
	grpc, err := cr.GetActiveGrpc()
	if err != nil {
		panic(err)
	}

	return grpc
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

func (cr *ChainRegistry) MustGetActiveWebSocket() string {
	rpc := cr.MustGetActiveRpc()
	websocket, err := normalizeRPCToWebSocket(rpc)
	if err != nil {
		panic(err)
	}
	return websocket
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

func (cr *ChainRegistry) MustGetDefaultFeeToken() FeeTokens {
	feeToken, err := cr.GetDefaultFeeToken()
	if err != nil {
		panic(err)
	}

	return feeToken
}

func (cr *ChainRegistry) MustGetDefaultMinGasPrices() string {
	feeToken := cr.MustGetDefaultFeeToken()

	return fmt.Sprintf("%s%s", strconv.FormatFloat(feeToken.FixedMinGasPrice, 'f', -1, 64), feeToken.Denom)
}

func loadChainRegistry(chainType ChainType) error {
	httpClient := client.NewHTTPClient()
	endpoint := GetRegistryEndpoint(chainType)
	LoadedChainRegistry[chainType] = &ChainRegistry{}
	if _, err := httpClient.Get(endpoint, "", nil, LoadedChainRegistry[chainType]); err != nil {
		return err
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

func MustGetChainRegistry(chainType ChainType) *ChainRegistry {
	chainRegistry, err := GetChainRegistry(chainType)
	if err != nil {
		panic(err)
	}

	return chainRegistry
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

func MustGetL2Registry(chainType ChainType, chainId string) *ChainRegistry {
	registry, err := GetL2Registry(chainType, chainId)
	if err != nil {
		panic(err)
	}

	return registry
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

	if len(networks) == 0 {
		return nil, fmt.Errorf("no chains found for chain type %s", chainType)
	}

	return networks, nil
}

func MustGetAllL2AvailableNetwork(chainType ChainType) []L2AvailableNetwork {
	networks, err := GetAllL2AvailableNetwork(chainType)
	if err != nil {
		panic(err)
	}

	return networks
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

func MustGetOPInitBotsSpecVersion(chainId string) int {
	version, err := GetOPInitBotsSpecVersion(chainId)
	if err != nil {
		panic(err)
	}

	return version
}

func (cr *ChainRegistry) MustGetCounterpartyClientId(portID, channelID string) Connection {
	address := cr.MustGetActiveLcd()
	httpClient := client.NewHTTPClient()

	var channel Channel
	if _, err := httpClient.Get(address, fmt.Sprintf("/ibc/core/channel/v1/channels/%s/ports/%s", channelID, portID), nil, &channel); err != nil {
		panic(err)
	}

	var connection Connection
	if _, err := httpClient.Get(address, fmt.Sprintf("/ibc/core/connection/v1/connections/%s", channel.Channel.ConnectionHops[0]), nil, &connection); err != nil {
		panic(err)
	}

	return connection
}
