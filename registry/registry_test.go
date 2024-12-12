package registry

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
)

func TestLoadChainRegistry(t *testing.T) {
	err := loadChainRegistry(CelestiaTestnet)
	if err != nil {
		t.Errorf("LoadChainRegistry() error for %s = %v", CelestiaTestnet, err)
	}

	loadedRegistry := LoadedChainRegistry[CelestiaTestnet]
	if loadedRegistry == nil {
		t.Fatal("expected chain registry to be loaded but got nil")
	}

	err = loadChainRegistry(CelestiaMainnet)
	if err != nil {
		t.Errorf("LoadChainRegistry() error for %s = %v", CelestiaMainnet, err)
	}

	loadedRegistry = LoadedChainRegistry[CelestiaMainnet]
	if loadedRegistry == nil {
		t.Fatal("expected chain registry to be loaded but got nil")
	}

	err = loadChainRegistry(InitiaL1Testnet)
	if err != nil {
		t.Errorf("LoadChainRegistry() error for %s = %v", InitiaL1Testnet, err)
	}

	loadedRegistry = LoadedChainRegistry[InitiaL1Testnet]
	if loadedRegistry == nil {
		t.Fatal("expected chain registry to be loaded but got nil")
	}
}

func TestGetChainRegistry(t *testing.T) {
	registry, err := GetChainRegistry(CelestiaMainnet)
	if err != nil {
		t.Errorf("GetChainRegistry() error = %v", err)
	}

	if registry.Bech32Prefix == "" {
		t.Errorf("invalid bech32 prefix")
	}
}

func TestLoadOPInitBotsSpecVersion(t *testing.T) {
	err := loadOPInitBotsSpecVersion()

	if err != nil {
		t.Errorf("LoadOPInitBotsSpecVersion() error = %v", err)
	}

	if OPInitBotsSpecVersion == nil {
		t.Error("expected OPInitBotsSpecVersion to be loaded")
	}
}

func TestGetOPInitBotsSpecVersion(t *testing.T) {
	tests := []struct {
		name    string
		chainId string
		want    int
		wantErr bool
	}{
		{
			name:    "successfully retrieve version",
			chainId: "initiation-2",
			want:    1,
			wantErr: false,
		},
		{
			name:    "chain id not found",
			chainId: "initiation-1",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetOPInitBotsSpecVersion(tt.chainId)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOPInitBotsSpecVersion() error = %v, wantErr %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("GetOPInitBotsSpecVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test GetMinGasPriceByDenom
func TestGetMinGasPriceByDenom(t *testing.T) {
	cr := ChainRegistry{
		Fees: Fees{
			FeeTokens: []FeeTokens{
				{Denom: "uinit", FixedMinGasPrice: 0.01},
				{Denom: "umin", FixedMinGasPrice: 0.02},
			},
		},
	}

	tests := []struct {
		denom     string
		expected  string
		expectErr bool
	}{
		{"uinit", "0.01uinit", false},
		{"umin", "0.02umin", false},
		{"btc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.denom, func(t *testing.T) {
			result, err := cr.GetMinGasPriceByDenom(tt.denom)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if result != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}

// Test GetActiveRpc
func TestGetActiveRpc(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cr := ChainRegistry{
		Apis: Apis{
			Rpc: []Endpoint{
				{Address: "http://invalid.rpc"}, // This will fail.
				{Address: server.URL},
			},
		},
	}

	result, err := cr.GetActiveRpc()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != server.URL {
		t.Errorf("expected: %s, got: %s", server.URL, result)
	}
}

// Test GetActiveLcd
func TestGetActiveLcd(t *testing.T) {
	// Start a test server to simulate the /health endpoint.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cosmos/base/tendermint/v1beta1/syncing" {
			w.WriteHeader(http.StatusOK) // Simulate a healthy LCD.
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cr := ChainRegistry{
		Apis: Apis{
			Rest: []Endpoint{
				{Address: server.URL},
				{Address: "http://invalid.lcd"}, // This will fail.
			},
		},
	}

	result, err := cr.GetActiveLcd()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != server.URL {
		t.Errorf("expected: %s, got: %s", server.URL, result)
	}
}

// Test GetActiveGrpc
func TestGetActiveGrpc(t *testing.T) {
	// Start a test gRPC server to simulate the gRPC server being healthy.
	grpcServer := grpc.NewServer()

	// Start gRPC server in a separate goroutine
	lis, err := net.Listen("tcp", ":9091")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Errorf("failed to serve gRPC server: %v", err)
			return
		}
	}()
	defer grpcServer.Stop()

	// Simulating a chain registry with an invalid gRPC endpoint and a valid HTTP endpoint.
	cr := ChainRegistry{
		Apis: Apis{
			Grpc: []Endpoint{
				{Address: "http://invalid.grpc"}, // This will fail.
				{Address: "localhost:9091"},      // gRPC server will succeed.
			},
		},
	}

	// Call the method to get the active gRPC server.
	result, err := cr.GetActiveGrpc()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Assert that the result is the correct gRPC server URL.
	if result != "localhost:9091" {
		t.Errorf("expected: %s, got: %s", "localhost:9091", result)
	}
}

// Test GetSeeds
func TestGetSeeds(t *testing.T) {
	cr1 := ChainRegistry{
		Peers: Peers{
			Seeds: []Peer{
				{Id: "seed1", Address: "192.168.1.1"},
				{Id: "seed2", Address: "192.168.1.2"},
			},
		},
	}

	expected1 := "seed1@192.168.1.1,seed2@192.168.1.2"
	result1 := cr1.GetSeeds()
	if result1 != expected1 {
		t.Errorf("expected: %s, got: %s", expected1, result1)
	}

	cr2 := ChainRegistry{
		Peers: Peers{
			Seeds: []Peer{},
		},
	}

	expected2 := ""
	result2 := cr2.GetSeeds()
	if result2 != expected2 {
		t.Errorf("expected: %s, got: %s", expected2, result2)
	}
}

// Test GetPersistentPeers
func TestGetPersistentPeers(t *testing.T) {
	cr1 := ChainRegistry{
		Peers: Peers{
			PersistentPeers: []Peer{
				{Id: "peer1", Address: "10.0.0.1"},
				{Id: "peer2", Address: "10.0.0.2"},
			},
		},
	}

	expected1 := "peer1@10.0.0.1,peer2@10.0.0.2"
	result1 := cr1.GetPersistentPeers()
	if result1 != expected1 {
		t.Errorf("expected: %s, got: %s", expected1, result1)
	}

	cr2 := ChainRegistry{
		Peers: Peers{
			PersistentPeers: []Peer{
				{Id: "peer1", Address: "10.0.0.1"},
				{Id: "peer2", Address: "10.0.0.2"},
			},
		},
	}

	expected2 := "peer1@10.0.0.1,peer2@10.0.0.2"
	result2 := cr2.GetPersistentPeers()
	if result2 != expected2 {
		t.Errorf("expected: %s, got: %s", expected2, result2)
	}
}

func TestGetChainId(t *testing.T) {
	tests := []struct {
		name     string
		input    ChainRegistry
		expected string
	}{
		{
			name:     "Valid Chain ID",
			input:    ChainRegistry{ChainId: "initiation-4"},
			expected: "initiation-4",
		},
		{
			name:     "Empty Chain ID",
			input:    ChainRegistry{ChainId: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.GetChainId()
			if result != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestGetBech32Prefix(t *testing.T) {
	tests := []struct {
		name     string
		input    ChainRegistry
		expected string
	}{
		{
			name:     "Valid bech32 prefix",
			input:    ChainRegistry{Bech32Prefix: "init"},
			expected: "init",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.GetBech32Prefix()
			if result != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestCheckAndAddPort(t *testing.T) {
	tests := []struct {
		address  string
		expected string
	}{
		{"https://example.com", "https://example.com:443"},
		{"http://example.com", "http://example.com:80"},
		{"https://example.com:26657", "https://example.com:26657"},
		{"http://example.com:443", "http://example.com:443"},
	}

	for _, test := range tests {
		result, err := checkAndAddPort(test.address)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result != test.expected {
			t.Errorf("For address %s, expected %s, got %s", test.address, test.expected, result)
		}
	}
}

const (
	MinimoveChainId string = "minimove-2"
	MiniwasmChainId string = "miniwasm-2"
	MinievmChainId  string = "minievm-2"
)

func TestLoadL2Registry(t *testing.T) {
	err := loadL2RegistryForType(InitiaL1Testnet)
	if err != nil {
		t.Errorf("loadL2RegistryForType() error for %s = %v", InitiaL1Testnet, err)
	}

	loadedRegistry := LoadedL2Registry[MinimoveChainId]
	if loadedRegistry == nil {
		t.Fatal("expected chain registry to be loaded but got nil")
	}

	loadedRegistry = LoadedL2Registry[MiniwasmChainId]
	if loadedRegistry == nil {
		t.Fatal("expected chain registry to be loaded but got nil")
	}

	loadedRegistry = LoadedL2Registry[MinievmChainId]
	if loadedRegistry == nil {
		t.Fatal("expected chain registry to be loaded but got nil")
	}
}

func TestGetL2Registry(t *testing.T) {
	registry, err := GetL2Registry(InitiaL1Testnet, MinimoveChainId)
	if err != nil {
		t.Errorf("GetL2Registry() error = %v", err)
	}

	if registry.Bech32Prefix == "" {
		t.Errorf("invalid bech32 prefix")
	}
}
