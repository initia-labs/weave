package client

import (
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
)

type testReflectionServer struct {
	grpc_reflection_v1.UnimplementedServerReflectionServer
}

func (s *testReflectionServer) ServerReflectionInfo(stream grpc_reflection_v1.ServerReflection_ServerReflectionInfoServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if req == nil {
			continue
		}

		resp := &grpc_reflection_v1.ServerReflectionResponse{
			ValidHost: req.Host,
			MessageResponse: &grpc_reflection_v1.ServerReflectionResponse_ListServicesResponse{
				ListServicesResponse: &grpc_reflection_v1.ListServiceResponse{
					Service: []*grpc_reflection_v1.ServiceResponse{},
				},
			},
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// Test the CheckHealth method with a mock gRPC server
func TestGRPCClient_CheckHealth_Success(t *testing.T) {
	server := grpc.NewServer()
	grpc_reflection_v1.RegisterServerReflectionServer(server, &testReflectionServer{})

	lis, err := net.Listen("tcp", "127.0.0.1:0")
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

	serverAddr := lis.Addr().String()

	client := NewGRPCClient()
	err = client.CheckHealth(serverAddr)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	assert.NoError(t, err)
}
