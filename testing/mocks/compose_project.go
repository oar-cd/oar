// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/oar-cd/oar/services"
	"github.com/stretchr/testify/mock"
)

// MockComposeProject implements ComposeProjectInterface for testing
type MockComposeProject struct {
	mock.Mock
}

func (m *MockComposeProject) Up(startServices bool) (string, string, error) {
	args := m.Called(startServices)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockComposeProject) Down(removeVolumes bool) (string, string, error) {
	args := m.Called(removeVolumes)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockComposeProject) Logs() (string, string, error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockComposeProject) GetConfig() (string, string, error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockComposeProject) Pull() (string, string, error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockComposeProject) Build() (string, string, error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockComposeProject) InitializeVolumeMounts() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockComposeProject) Status() (*services.ComposeStatus, error) {
	args := m.Called()
	return args.Get(0).(*services.ComposeStatus), args.Error(1)
}

func (m *MockComposeProject) UpStreaming(startServices bool, outputChan chan<- services.StreamMessage) error {
	args := m.Called(startServices, outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) UpPiping(startServices bool) error {
	args := m.Called(startServices)
	return args.Error(0)
}

func (m *MockComposeProject) DownStreaming(outputChan chan<- services.StreamMessage) error {
	args := m.Called(outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) DownPiping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockComposeProject) LogsStreaming(ctx context.Context, outputChan chan<- string) error {
	args := m.Called(ctx, outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) LogsPiping() error {
	args := m.Called()
	return args.Error(0)
}
