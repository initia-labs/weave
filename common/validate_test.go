package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateURL(t *testing.T) {
	failTests := []struct {
		input string
	}{
		{"//localhost:26657"},
		{"localhost:26657"},
		{"ws://localhost:26657/websocket"},
		{"wss://localhost:26657/websocket"},
	}

	for _, test := range failTests {
		err := ValidateURL(test.input)
		if err == nil {
			t.Errorf("For input '%s', expected error, but got nil", test.input)
		}
	}

	successTests := []struct {
		input string
	}{
		{"http://localhost:26657"},
		{"https://localhost:26657"},
		{"https://localhost:26657/abc"},
	}

	for _, test := range successTests {
		err := ValidateURL(test.input)
		if err != nil {
			t.Errorf("For input '%s', expected no error, but got '%v'", test.input, err)
		}
	}
}

func TestValidateWSURL(t *testing.T) {
	failTests := []struct {
		input string
	}{
		{"http://localhost:26657"},
		{"https://localhost:26657"},
	}

	for _, test := range failTests {
		err := ValidateWSURL(test.input)
		if err == nil {
			t.Errorf("For input '%s', expected error, but got nil", test.input)
		}
	}

	successTests := []struct {
		input string
	}{
		{"ws://localhost:26657/websocket"},
		{"wss://localhost:26657/websocket"},
	}

	for _, test := range successTests {
		err := ValidateWSURL(test.input)
		if err != nil {
			t.Errorf("For input '%s', expected no error, but got '%v'", test.input, err)
		}
	}
}

func TestValidatePositiveBigIntOrZero(t *testing.T) {
	maxUint256 := strings.Repeat("9", 78) // approximate max uint256 value

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid positive integer",
			input:   "123",
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   "-123",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "decimal number",
			input:   "12.3",
			wantErr: true,
		},
		{
			name:    "exceeds uint256",
			input:   maxUint256 + "0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositiveBigIntOrZero(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePositiveBigIntOrZero() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePositiveBigInt(t *testing.T) {
	maxUint256 := strings.Repeat("9", 78) // approximate max uint256 value

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid positive integer",
			input:   "123",
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			wantErr: true,
		},
		{
			name:    "negative integer",
			input:   "-123",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "exceeds uint256",
			input:   maxUint256 + "0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositiveBigInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePositiveBigInt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURLWithPort(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid http URL with port",
			url:     "http://example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid https URL with port",
			url:     "https://example.com:443",
			wantErr: false,
		},
		{
			name:    "missing port",
			url:     "http://example.com",
			wantErr: true,
			errMsg:  "URL must include a port number",
		},
		{
			name:    "invalid port (too high)",
			url:     "http://example.com:65536",
			wantErr: true,
			errMsg:  "invalid port number: must be between 1 and 65535",
		},
		{
			name:    "invalid port (zero)",
			url:     "http://example.com:0",
			wantErr: true,
			errMsg:  "invalid port number: must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURLWithPort(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
