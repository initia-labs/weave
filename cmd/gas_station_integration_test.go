//go:build integration
// +build integration

package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/initia-labs/weave/common"
	"github.com/initia-labs/weave/models"
	"github.com/initia-labs/weave/service"
	"github.com/initia-labs/weave/testutil"
)

func TestGasStationSetup(t *testing.T) {
	setup(t, []service.Command{})
	defer teardown(t, []service.Command{})

	userHome, _ := os.UserHomeDir()
	weaveDir := filepath.Join(userHome, common.WeaveDirectory)

	finalModel := testutil.SetupGasStation(t)

	// Check the final state here
	assert.IsType(t, &models.WeaveAppSuccessfullyInitialized{}, finalModel)

	if _, ok := finalModel.(*models.WeaveAppSuccessfullyInitialized); ok {
		assert.True(t, ok)
	}

	// Check if Weave home has been created
	_, err := os.Stat(weaveDir)
	assert.Nil(t, err)

	// Assert values
	weaveConfig := filepath.Join(weaveDir, "config.json")
	testutil.CompareJsonValue(t, weaveConfig, "common.gas_station.mnemonic", testutil.GasStationMnemonic)
}
