package weaveinit

import (
	"testing"

	"github.com/initia-labs/weave/flags"
)

func TestGetWeaveInitOptions(t *testing.T) {
	// Test with different flag combinations
	testCases := []struct {
		flags    string
		expected []WeaveInitOption
	}{
		{"", []WeaveInitOption{RunL1NodeOption}},
		{"minitia_launch", []WeaveInitOption{RunL1NodeOption, LaunchNewMinitiaOption}},
		{"minitia_launch,opinit_bots", []WeaveInitOption{RunL1NodeOption, LaunchNewMinitiaOption, SetupOPBotsKeys, InitializeOPBotsOption}},
		{"minitia_launch,opinit_bots,relayer", []WeaveInitOption{RunL1NodeOption, LaunchNewMinitiaOption, SetupOPBotsKeys, InitializeOPBotsOption, StartRelayerOption}},
	}

	for _, tc := range testCases {
		flags.EnabledFlags = tc.flags
		options := GetWeaveInitOptions()

		if len(options) != len(tc.expected) {
			t.Errorf("For flags '%s': Expected %d options, but got %d", tc.flags, len(tc.expected), len(options))
		}

		for i, option := range options {
			if option != tc.expected[i] {
				t.Errorf("For flags '%s': Expected option %d to be %s, but got %s", tc.flags, i, tc.expected[i], option)
			}
		}
	}
}
