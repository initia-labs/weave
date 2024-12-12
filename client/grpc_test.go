package client

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockServerReflectionClient is a mock of the grpc_reflection_v1.ServerReflectionServer interface.
type MockServerReflectionClient struct {
	mock.Mock
}

// MockServerReflectionInfoServer is the mock server stream for ServerReflectionInfo.
type MockServerReflectionInfoServer struct {
	mock.Mock
	grpc.ServerStream
}

func (m *MockServerReflectionInfoServer) SendMsg(mess interface{}) error {
	args := m.Called(mess)
	return args.Error(0)
}

func (m *MockServerReflectionInfoServer) RecvMsg(mess interface{}) error {
	args := m.Called(mess)
	return args.Error(0)
}

// Test the CheckHealth method with a mock gRPC server
func TestGRPCClient_CheckHealth_Success(t *testing.T) {
	mockServerStream := new(MockServerReflectionInfoServer)
	mockClient := new(MockServerReflectionClient)

	mockClient.On("ServerReflectionInfo", mockServerStream).Return(nil)
	mockServerStream.On("SendMsg", mock.Anything).Return(nil)
	mockServerStream.On("RecvMsg", mock.Anything).Return(nil)

	server := grpc.NewServer()

	lis, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		t.Fatalf("failed to listen on port: %v", err)
	}

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("failed to serve mock gRPC server: %v", err)
			return
		}
	}()
	defer server.Stop()

	serverAddr := "localhost:9090"

	client := NewGRPCClient()
	err = client.CheckHealth(serverAddr)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	assert.NoError(t, err)
}
