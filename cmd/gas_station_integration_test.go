//go:build integration
// +build integration

package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/analytics"
	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/models"
	"github.com/initia-labs/weave/testutil"
)

func TestMain(m *testing.M) {
	analytics.Client = &analytics.NoOpClient{}
}

func TestGasStationSetup(t *testing.T) {
	finalModel := testutil.SetupGasStation(t)

	// Check the final state here
	assert.IsType(t, &models.WeaveAppSuccessfullyInitialized{}, finalModel)

	if _, ok := finalModel.(*models.WeaveAppSuccessfullyInitialized); ok {
		assert.True(t, ok)
	}

	// Check if Weave home has been created
	userHome, _ := os.UserHomeDir()
	weaveDir := filepath.Join(userHome, common.WeaveDirectory)
	_, err := os.Stat(weaveDir)
	assert.Nil(t, err)

	// Assert values
	weaveConfig := filepath.Join(weaveDir, "config.json")
	testutil.CompareJsonValue(t, weaveConfig, "common.gas_station_mnemonic", testutil.GasStationMnemonic)
}
