package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/initia-labs/weave/config"
	"github.com/initia-labs/weave/context"
	"github.com/initia-labs/weave/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
)

const (
	DefaultMaxWaitRetry   = 300
	DefaultPostWaitPeriod = 5 * time.Second
	GasStationMnemonic    = "imitate sick vibrant bonus weather spice pave announce direct impulse strategy math"
)

type Step interface {
	Execute(prog tea.Program)
	Wait() bool
}

type Steps []Step

type InputStep struct {
	Msg tea.Msg
}

func (i InputStep) Execute(prog tea.Program) {
	prog.Send(i.Msg)
}

func (i InputStep) Wait() bool {
	return true
}

type WaitStep struct {
	Check func() bool
}

func (w WaitStep) Execute(_ tea.Program) {}

func (w WaitStep) Wait() bool {
	return w.Check()
}

func RunProgramWithSteps(t *testing.T, program tea.Model, steps Steps) tea.Model {
	prog := tea.NewProgram(program, tea.WithInput(nil))
	done := make(chan struct{})
	finalModel := tea.Model(nil)

	go func() {
		var err error
		finalModel, err = prog.Run()
		if err != nil {
			t.Errorf("Program run failed: %v", err)
			return
		}
		close(done)
	}()

	for _, step := range steps {
		if waitStep, ok := step.(WaitStep); ok {
			retryCount := 0
			for {
				if waitStep.Wait() {
					break
				}

				if retryCount >= DefaultMaxWaitRetry {
					t.Errorf("Max retries reached while waiting for condition in WaitStep")
					return nil
				}

				retryCount++
				time.Sleep(100 * time.Millisecond)
			}
			time.Sleep(DefaultPostWaitPeriod)
		}

		step.Execute(*prog)
		time.Sleep(100 * time.Millisecond)
	}

	<-done
	return finalModel
}

func ClearTestDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		panic(fmt.Sprintf("failed to remove test directory: %v", err))
	}
}

func GetTomlValue(filePath, key string) (interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var tomlData map[string]interface{}
	if err := toml.Unmarshal(data, &tomlData); err != nil {
		return nil, err
	}

	parts := strings.Split(key, ".")
	current := tomlData

	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part], nil
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			return nil, nil
		}
		current = next
	}

	return nil, nil
}

func CompareTomlValue(t *testing.T, filePath, key string, expectedValue interface{}) {
	value, err := GetTomlValue(filePath, key)
	assert.NoError(t, err, "Error loading TOML file or traversing key")

	assert.Equal(t, expectedValue, value, "Mismatch for key %s", key)
}

func GetJsonValue(filePath, key string) (interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	parts := strings.Split(key, ".")
	current := jsonData

	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part], nil
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			return nil, nil
		}
		current = next
	}

	return nil, nil
}

func CompareJsonValue(t *testing.T, filePath, key string, expectedValue interface{}) {
	value, err := GetJsonValue(filePath, key)
	assert.NoError(t, err, "Error loading JSON file or traversing key")

	assert.Equal(t, expectedValue, value, "Mismatch for key %s", key)
}

func SetupGasStation(t *testing.T) tea.Model {
	err := config.InitializeConfig()
	assert.Nil(t, err)

	ctx := context.NewAppContext(models.NewExistingCheckerState())
	firstModel := models.NewGasStationMnemonicInput(ctx)

	steps := Steps{
		TypeText(GasStationMnemonic),
		PressEnter,
	}

	return RunProgramWithSteps(t, firstModel, steps)
}
