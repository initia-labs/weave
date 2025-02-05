package common

import (
	"strings"
	"testing"
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

func TestValidateBigInt(t *testing.T) {
	// Create a string that's way too long to be a valid number
	tooLongNumber := strings.Repeat("9", 1000000)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid integer",
			input:   "123",
			wantErr: false,
		},
		{
			name:    "valid negative integer",
			input:   "-123",
			wantErr: false,
		},
		{
			name:    "valid zero",
			input:   "0",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			input:   "12a3",
			wantErr: true,
		},
		{
			name:    "decimal number",
			input:   "12.3",
			wantErr: true,
		},
		{
			name:    "extremely large number",
			input:   tooLongNumber,
			wantErr: true, // big.Int can handle arbitrary precision
		},
		{
			name:    "max uint64 + 1",
			input:   "18446744073709551616", // 2^64
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBigInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBigInt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePositiveBigInt(t *testing.T) {
	// Create a string that's way too long to be a valid number
	tooLongNumber := strings.Repeat("9", 1000000)

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
			name:    "invalid characters",
			input:   "12a3",
			wantErr: true,
		},
		{
			name:    "decimal number",
			input:   "12.3",
			wantErr: true,
		},
		{
			name:    "large valid number",
			input:   "123456789012345678901234567890",
			wantErr: false,
		},
		{
			name:    "extremely large number",
			input:   tooLongNumber,
			wantErr: true,
		},
		{
			name:    "max uint64 + 1",
			input:   "18446744073709551616", // 2^64
			wantErr: false,
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
