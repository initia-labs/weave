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
			hrp:       "init",
			expected:  "init1jvk3gadm45cxxg4g8y3c64h7thy3s36yat0ezy",
		},
		{
			name:      "Valid init address without 0x prefix (2)",
			pubKeyHex: "0x552bfcf61b41b22eab0a520b896b072a1cd22b8c",
			hrp:       "init",
			expected:  "init1254leasmgxeza2c22g9cj6c89gwdy2uvwv05qu",
		},
		{
			name:      "Valid init address without 0x prefix",
			pubKeyHex: "932d1475bbad306322a839238d56fe5dc9184744",
			hrp:       "init",
			expected:  "init1jvk3gadm45cxxg4g8y3c64h7thy3s36yat0ezy",
		},
		{
			name:      "Valid init capital address with 0x prefix ",
			pubKeyHex: "0x932D1475BBAD306322A839238D56FE5DC9184744",
			hrp:       "init",
			expected:  "init1jvk3gadm45cxxg4g8y3c64h7thy3s36yat0ezy",
		},
		{
			name:      "Invalid init address with 0x prefix",
			pubKeyHex: "0x932D1475BBAD306322A839238D56FE5DC91847441",
			hrp:       "init",
			expectErr: true,
		},
		{
			name:      "Invalid init address without 0x prefix",
			pubKeyHex: "932D1475BBAD306322A839238D56FE5DC91847441",
			hrp:       "init",
			expectErr: true,
		},
		{
			name:      "Invalid init address",
			pubKeyHex: "invalid",
			hrp:       "init",
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
