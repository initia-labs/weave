package client

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
)

const (
	DefaultTimeout = 3 * time.Second
)

// GRPCClient defines the logic for making gRPC requests.
type GRPCClient struct{}

// NewGRPCClient initializes and returns a new GRPCClient instance.
func NewGRPCClient() *GRPCClient {
	return &GRPCClient{}
}

// CheckHealth attempts to connect to the server and uses the reflection service to verify the server is up.
func (g *GRPCClient) CheckHealth(serverAddr string) error {
	serverAddr = strings.TrimPrefix(serverAddr, "grpc://")

	_, port, err := net.SplitHostPort(serverAddr)
	if err != nil {
		return fmt.Errorf("invalid grpc server address: %w", err)
	}

	var opts []grpc.DialOption
	if port == "443" {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(serverAddr, opts...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client (grpc.NewClient): %w; note: actual connection occurs later on the first RPC call", err)
	}
	defer conn.Close()

	reflectionClient := grpc_reflection_v1.NewServerReflectionClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	stream, err := reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to start reflection stream: %v", err)
	}

	req := &grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	}

	if err = stream.Send(req); err != nil {
		return fmt.Errorf("failed to send reflection request: %v", err)
	}

	return nil
}
