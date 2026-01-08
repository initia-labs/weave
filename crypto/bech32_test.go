package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPubKeyToBech32Address(t *testing.T) {
	tests := []struct {
		name      string
		pubKeyHex string
		hrp       string
		expected  string
		expectErr bool
	}{
		{
			name:      "Valid init address with 0x prefix (1)",
			pubKeyHex: "0x932d1475bbad306322a839238d56fe5dc9184744",
			expected:  "init1jvk3gadm45cxxg4g8y3c64h7thy3s36yat0ezy",
		},
		{
			name:      "Valid init address without 0x prefix (2)",
			pubKeyHex: "0x552bfcf61b41b22eab0a520b896b072a1cd22b8c",
			expected:  "init1254leasmgxeza2c22g9cj6c89gwdy2uvwv05qu",
		},
		{
			name:      "Valid init address without 0x prefix",
			pubKeyHex: "932d1475bbad306322a839238d56fe5dc9184744",
			expected:  "init1jvk3gadm45cxxg4g8y3c64h7thy3s36yat0ezy",
		},
		{
			name:      "Valid init capital address with 0x prefix ",
			pubKeyHex: "0x932D1475BBAD306322A839238D56FE5DC9184744",
			expected:  "init1jvk3gadm45cxxg4g8y3c64h7thy3s36yat0ezy",
		},
		{
			name:      "Invalid init address",
			pubKeyHex: "invalid",
			expectErr: true,
		},
		{
			name:      "Valid init address with 0x prefix (1)",
			pubKeyHex: "0x1",
			expected:  "init1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpqr5e3d",
		},
		{
			name:      "Valid init address with 0x prefix (2)",
			pubKeyHex: "0x6baa5dcfd050e9b85a4ddf214baee77884773ba4",
			expected:  "init1dw49mn7s2r5mskjdmus5hth80zz8wwaywycq06",
		},
		{
			name:      "Valid init address with 0x prefix (3)",
			pubKeyHex: "0x628d39fde7251e7ce340a82b73006aeb1b927238cba83322ee9ce8b892b2bb55",
			expected:  "init1v2xnnl08y508ec6q4q4hxqr2avdeyu3cew5rxghwnn5t3y4jhd2smmah7n",
		},
		{
			name:      "Valid init address with 0x prefix (3)",
			pubKeyHex: "0x628d39fde7251e7ce340a82b73006aeb1b927238cba83322ee9ce8b892b2bb55asdasdasdasdasdasdasdasdasdasdasda",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bech32Addr, err := PubKeyToBech32Address(tt.pubKeyHex)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, bech32Addr, "Test case: %s", tt.name)
			}
		})
	}
}

func TestMnemonicToBech32Address(t *testing.T) {
	tests := []struct {
		name        string
		hrp         string
		mnemonic    string
		addressType AddressType
		expected    string
		expectErr   bool
	}{
		// Cosmos address type tests
		{
			name:        "Cosmos address - standard test vector",
			hrp:         "init",
			mnemonic:    "imitate sick vibrant bonus weather spice pave announce direct impulse strategy math",
			addressType: CosmosAddressType,
			expected:    "init16pawh0v7w996jrmtzugz3hmhq0wx6ndq5pp0dr",
		},
		// EVM address type tests
		{
			name:        "EVM address - standard test vector",
			hrp:         "init",
			mnemonic:    "imitate sick vibrant bonus weather spice pave announce direct impulse strategy math",
			addressType: EVMAddressType,
			expected:    "init1rv0kk09mcus8nj6ffmgmw2ulsktgnepnjd2lp7",
		},
		// Error cases
		{
			name:        "Invalid mnemonic - too short",
			hrp:         "init",
			mnemonic:    "abandon abandon abandon",
			addressType: CosmosAddressType,
			expectErr:   true,
		},
		{
			name:        "Invalid mnemonic - invalid words",
			hrp:         "init",
			mnemonic:    "invalid words here that do not form a valid mnemonic phrase at all",
			addressType: CosmosAddressType,
			expectErr:   true,
		},
		{
			name:        "Empty mnemonic",
			hrp:         "init",
			mnemonic:    "",
			addressType: CosmosAddressType,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address, err := MnemonicToBech32Address(tt.hrp, tt.mnemonic, tt.addressType)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Empty(t, address)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, address)

				// Verify address has the correct HRP prefix
				assert.Contains(t, address, tt.hrp)

				if tt.expected != "" {
					assert.Equal(t, tt.expected, address)
				}
			}
		})
	}
}
