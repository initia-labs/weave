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
