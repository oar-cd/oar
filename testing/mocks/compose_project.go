// Package mocks provides mock implementations for testing.
package mocks

import (
	"github.com/ch00k/oar/services"
	"github.com/stretchr/testify/mock"
)

// MockComposeProject implements ComposeProjectInterface for testing
type MockComposeProject struct {
	mock.Mock
}

func (m *MockComposeProject) Up() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockComposeProject) Down() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockComposeProject) Logs() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockComposeProject) GetConfig() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockComposeProject) Status() (*services.ComposeStatus, error) {
	args := m.Called()
	return args.Get(0).(*services.ComposeStatus), args.Error(1)
}

func (m *MockComposeProject) UpStreaming(outputChan chan<- string) error {
	args := m.Called(outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) UpPiping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockComposeProject) DownStreaming(outputChan chan<- string) error {
	args := m.Called(outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) DownPiping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockComposeProject) LogsStreaming(outputChan chan<- string) error {
	args := m.Called(outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) LogsPiping() error {
	args := m.Called()
	return args.Error(0)
}
