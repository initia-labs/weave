package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/initia-labs/weave/common"
)

// Mocking the os package functions
type MockedFilesystem struct {
	mock.Mock
}

func (m *MockedFilesystem) UserHomeDir() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockedFilesystem) MkdirAll(path string, perm os.FileMode) error {
	args := m.Called(path, perm)
	return args.Error(0)
}

func (m *MockedFilesystem) Create(path string) (*os.File, error) {
	args := m.Called(path)
	return nil, args.Error(1)
}

func (m *MockedFilesystem) Stat(name string) (os.FileInfo, error) {
	args := m.Called(name)
	return nil, args.Error(1)
}

func TestInitializeConfig(t *testing.T) {
	fs := new(MockedFilesystem)
	home := "/mock/home"
	configPath := filepath.Join(home, common.WeaveConfigFile)

	// Resetting mocks for next test case
	fs.Mock = mock.Mock{}

	// Case 3: Successful configuration initialization
	fs.On("UserHomeDir").Return(home, nil)
	fs.On("MkdirAll", filepath.Dir(configPath), os.ModePerm).Return(nil)
	fs.On("Stat", configPath).Return(nil, errors.New("file does not exist"))
	fs.On("Create", configPath).Return(nil, nil)

	err := InitializeConfig()
	assert.NoError(t, err, "should initialize config without error")
}
