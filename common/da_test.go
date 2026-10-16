package common

import (
	"testing"

	"github.com/initia-labs/weave/internal/features"
	"github.com/stretchr/testify/require"
)

// TestDAPolicyTarget catches accidental Celestia enablement and normalization or
// acceptance of unsupported rollup DA targets.
func TestDAPolicyTarget(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		wantErr      bool
		wantDisabled bool
	}{
		{name: "Initia", target: "INITIA"},
		{name: "Celestia", target: "CELESTIA", wantErr: !features.CelestiaDAEnabled, wantDisabled: !features.CelestiaDAEnabled},
		{name: "empty", target: "", wantErr: true},
		{name: "misspelled Celestia", target: "CELESTIAA", wantErr: true},
		{name: "trailing whitespace", target: "INITIA ", wantErr: true},
		{name: "leading whitespace", target: " INITIA", wantErr: true},
		{name: "mixed case", target: "Initia", wantErr: true},
		{name: "lowercase Celestia", target: "celestia", wantErr: true},
		{name: "prefix", target: "init", wantErr: true},
		{name: "unknown", target: "UNKNOWN", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNewDATarget(tt.target)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.wantDisabled {
				require.ErrorIs(t, err, ErrCelestiaDADisabled)
			} else {
				require.NotErrorIs(t, err, ErrCelestiaDADisabled)
			}
		})
	}
}

// TestDAPolicyPrefix catches accidental Celestia enablement and normalization or
// acceptance of unsupported executor DA address prefixes.
func TestDAPolicyPrefix(t *testing.T) {
	tests := []struct {
		name         string
		prefix       string
		wantErr      bool
		wantDisabled bool
	}{
		{name: "Initia", prefix: "init"},
		{name: "Celestia", prefix: "celestia", wantErr: !features.CelestiaDAEnabled, wantDisabled: !features.CelestiaDAEnabled},
		{name: "empty", prefix: "", wantErr: true},
		{name: "misspelled Celestia", prefix: "celestiaa", wantErr: true},
		{name: "trailing whitespace", prefix: "init ", wantErr: true},
		{name: "leading whitespace", prefix: " init", wantErr: true},
		{name: "mixed case", prefix: "Init", wantErr: true},
		{name: "target", prefix: "CELESTIA", wantErr: true},
		{name: "unknown", prefix: "cosmos", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNewDAPrefix(tt.prefix)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.wantDisabled {
				require.ErrorIs(t, err, ErrCelestiaDADisabled)
			} else {
				require.NotErrorIs(t, err, ErrCelestiaDADisabled)
			}
		})
	}
}
