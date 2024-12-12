package opinit_test

//const (
//	TestOPInitHome = ".opinit.weave.test"
//)

// func setupOPInitBots(t *testing.T) tea.Model {
// 	ctx := context.NewAppContext(opinit_bots.NewOPInitBotsState())
// 	ctx = context.SetOPInitHome(ctx, TestOPInitHome)

// 	versions, currentVersion := cosmosutils.GetOPInitVersions()
// 	firstModel := opinit_bots.NewOPInitBotVersionSelector(ctx, versions, currentVersion)

// 	// Ensure that there is no previous OPInit home
// 	_, err := os.Stat(TestOPInitHome)
// 	assert.NotNil(t, err)

// 	steps := []Step{
// 		pressEnter,
// 		pressEnter,
// 		pressSpace,
// 		pressDown,
// 		pressSpace,
// 		pressDown,
// 		pressSpace,
// 		pressDown,
// 		pressSpace,
// 		pressEnter,
// 		pressEnter,
// 		pressEnter,
// 		pressEnter,
// 		pressEnter,
// 		pressEnter,
// 	}

// 	return runProgramWithSteps(t, firstModel, steps)
// }

// func TestOPInitBotsSetup(t *testing.T) {
// 	finalModel := setupOPInitBots(t)

// 	// Check the final state here
// 	assert.IsType(t, &opinit_bots.TerminalState{}, finalModel)

// 	if _, ok := finalModel.(*opinit_bots.TerminalState); ok {
// 		assert.True(t, ok)
// 	}

// 	// Check if OPInit home has been created
// 	_, err := os.Stat(TestOPInitHome)
// 	assert.Nil(t, err)

// 	// Check the keys have been created
// 	_, err = os.Stat(filepath.Join(TestOPInitHome, "weave-dummy/keyring-test/weave_batch_submitter.info"))
// 	assert.Nil(t, err)

// 	_, err = os.Stat(filepath.Join(TestOPInitHome, "weave-dummy/keyring-test/weave_bridge_executor.info"))
// 	assert.Nil(t, err)

// 	_, err = os.Stat(filepath.Join(TestOPInitHome, "weave-dummy/keyring-test/weave_challenger.info"))
// 	assert.Nil(t, err)

// 	_, err = os.Stat(filepath.Join(TestOPInitHome, "weave-dummy/keyring-test/weave_output_submitter.info"))
// 	assert.Nil(t, err)

// 	ctx := context.NewAppContext(opinit_bots.NewOPInitBotsState())
// 	ctx = context.SetOPInitHome(ctx, TestOPInitHome)

// 	versions, currentVersion := cosmosutils.GetOPInitVersions()
// 	firstModel := opinit_bots.NewOPInitBotVersionSelector(ctx, versions, currentVersion)

// 	// Ensure that there is an existing OPInit home
// 	_, err = os.Stat(TestOPInitHome)
// 	assert.Nil(t, err)

// 	steps := []Step{
// 		pressEnter,
// 		pressEnter,
// 		pressSpace,
// 		pressDown,
// 		pressSpace,
// 		pressDown,
// 		pressSpace,
// 		pressDown,
// 		pressSpace,
// 		pressEnter,
// 		pressDown,
// 		pressEnter,
// 		typeText("false seek emerge venue park present color knock notice spike use notable"),
// 		pressEnter,
// 		pressDown,
// 		pressEnter,
// 		typeText("people assist noble flower turtle canoe sand wall useful accuse trash zone"),
// 		pressEnter,
// 		pressDown,
// 		pressEnter,
// 		typeText("delay brick cradle knock indoor squeeze enlist arrange smooth limit symbol south"),
// 		pressEnter,
// 		pressEnter,
// 		pressDown,
// 		pressEnter,
// 		typeText("educate protect return spirit finger region portion dish seven boost measure chase"),
// 		pressEnter,
// 	}

// 	finalModel = runProgramWithSteps(t, firstModel, steps)
// 	defer clearTestDir(TestOPInitHome)

// 	// Let's test the keys
// 	userHome, _ := os.UserHomeDir()
// 	opinitBinary := filepath.Join(userHome, common.WeaveDataDirectory, "opinitd")

// 	cmd := exec.Command(opinitBinary, "keys", "show", "weave-dummy", "weave_batch_submitter", "--home", TestOPInitHome)
// 	outputBytes, err := cmd.CombinedOutput()
// 	assert.Nil(t, err)
// 	assert.Equal(t, "weave_batch_submitter: init1wzccyequn0yqc5mne6yev6u628yvyfwpr7y38d\n", string(outputBytes), "Mismatch for key weave_batch_submitter, expected init1wzccyequn0yqc5mne6yev6u628yvyfwpr7y38d but got %s", string(outputBytes))

// 	cmd = exec.Command(opinitBinary, "keys", "show", "weave-dummy", "weave_bridge_executor", "--home", TestOPInitHome)
// 	outputBytes, err = cmd.CombinedOutput()
// 	assert.Nil(t, err)
// 	assert.Equal(t, "weave_bridge_executor: init1eul78cxrljfn47l0f7qpgue7l3p9pa9j86w6hq\n", string(outputBytes), "Mismatch for key weave_bridge_executor, expected init1eul78cxrljfn47l0f7qpgue7l3p9pa9j86w6hq but got %s", string(outputBytes))

// 	cmd = exec.Command(opinitBinary, "keys", "show", "weave-dummy", "weave_challenger", "--home", TestOPInitHome)
// 	outputBytes, err = cmd.CombinedOutput()
// 	assert.Nil(t, err)
// 	assert.Equal(t, "weave_challenger: init1masuevcdvkra3nr7p2dkwa8lq2hga75ym279tr\n", string(outputBytes), "Mismatch for key weave_challenger, expected init1masuevcdvkra3nr7p2dkwa8lq2hga75ym279tr but got %s", string(outputBytes))

// 	cmd = exec.Command(opinitBinary, "keys", "show", "weave-dummy", "weave_output_submitter", "--home", TestOPInitHome)
// 	outputBytes, err = cmd.CombinedOutput()
// 	assert.Nil(t, err)
// 	assert.Equal(t, "weave_output_submitter: init1kd3fc4407sgaakguj4scvhzdv6r907ncdetd94\n", string(outputBytes), "Mismatch for key weave_output_submitter, expected init1kd3fc4407sgaakguj4scvhzdv6r907ncdetd94 but got %s", string(outputBytes))
// }

//
//func TestOPInitBotsInit(t *testing.T) {
//	ctx := utils.NewAppContext(opinit_bots.NewOPInitBotsState())
//	ctx = utils.SetOPInitHome(ctx, TestOPInitHome)
//
//	firstModel := opinit_bots.NewOPInitBotInitSelector(ctx)
//
//	// Ensure that there is no previous OPInit home
//	_, err := os.Stat(TestOPInitHome)
//	assert.NotNil(t, err)
//
//	steps := []Step{
//		pressEnter,
//		pressEnter,
//		pressTab,
//		pressEnter,
//	}
//
//	finalModel := runProgramWithSteps(t, firstModel, steps)
//
//	// Check the final state here
//	assert.IsType(t, &opinit_bots.TerminalState{}, finalModel)
//
//	if _, ok := finalModel.(*opinit_bots.TerminalState); ok {
//		assert.True(t, ok)
//	}
//}
