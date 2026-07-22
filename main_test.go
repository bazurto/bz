package main

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/bazurto/bz/grpc/bazurto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

// mockVersionServer implements the Bazurto gRPC service in-process so the
// test does not depend on an external server.
type mockVersionServer struct {
	pb.UnimplementedBazurtoServer
	gotToken string
}

func (s *mockVersionServer) GetLatestVersion(ctx context.Context, req *pb.LatestVersionRequest) (*pb.LatestVersionResponse, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("xtoken"); len(v) > 0 {
			s.gotToken = v[0]
		}
	}
	return &pb.LatestVersionResponse{Version: "1.2.3"}, nil
}

func TestGrpcServer(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	mock := &mockVersionServer{}
	pb.RegisterBazurtoServer(srv, mock)
	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(
		metadata.NewOutgoingContext(context.Background(),
			metadata.New(map[string]string{"xtoken": "12345"}),
		), time.Second*5,
	)
	defer cancel()

	clientConn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer clientConn.Close()

	client := pb.NewBazurtoClient(clientConn)

	resp, err := client.GetLatestVersion(ctx, &pb.LatestVersionRequest{
		Owner: "bazurto",
		Repo:  "bz",
	})
	if err != nil {
		t.Fatalf("Error calling GetLatestVersion: %v", err)
	}

	if resp.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", resp.Version)
	}
	if mock.gotToken != "12345" {
		t.Errorf("expected xtoken metadata 12345, got %q", mock.gotToken)
	}
}
