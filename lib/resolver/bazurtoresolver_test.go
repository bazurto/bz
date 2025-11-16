package resolver

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"math/rand"

	pb "github.com/bazurto/bz/grpc/bazurto" // Adjust the import path as necessary
	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/utils"
	"google.golang.org/grpc"
)

func TestBazurtoResolver(t *testing.T) {
	tmpDir := t.TempDir()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomPort := r.Intn(65535-1024) + 1024
	appCtx := model.NewDefaultAppContext()
	appCtx.UserCacheDirName = tmpDir

	// create artifact
	work := path.Join(tmpDir, "work")
	bin := path.Join(work, "bin")
	os.MkdirAll(bin, 0755)
	os.WriteFile(path.Join(work, "bz.yaml"), []byte("name: test\n"), 0644)
	os.WriteFile(path.Join(bin, "executable"), []byte("#!/bin/sh\necho bz\n"), 0755)
	f, _ := os.Create(path.Join(tmpDir, "owner-repo-linux-amd64.zip"))
	utils.Zip(work, f, nil)
	defer f.Close()

	mockGrpcServer := startMockBazurtoGrpcServer(randomPort)
	defer mockGrpcServer.Stop()

	mockGrpcServer.ResolveCoordFunc = func(ctx context.Context, rcr *pb.ResolveCoordRequest) (*pb.ResolveCoordResponse, error) {
		return &pb.ResolveCoordResponse{
			Coord: "http://bazurto/owner/repo#1.2.3",
			Artifacts: []*pb.Artifact{
				&pb.Artifact{
					Os:   "linux",
					Arch: "amd64",
					Name: "owner-repo-linux-amd64.zip",
				},
				&pb.Artifact{
					Name: "owner-repo.zip",
				},
			},
		}, nil
	}

	mockGrpcServer.DownloadCoordFunc = func(req *pb.DownloadCoordRequest, srv grpc.ServerStreamingServer[pb.BinaryChunk]) {
	}

	resolver := NewBazurtoResolver(appCtx, WithHost("localhost"), WithPort(randomPort))

	c, _ := model.NewCoordFromStr("owner/repo#1.2.3")
	lc, err := resolver.ResolveCoord(c)
	if err != nil {
		t.Fatalf("Failed to resolve coord: %v", err)
	}
	if lc.URL.Hostname() != "bazurto" {
		t.Errorf("Expected URL host to be 'bazurto', got '%s'", lc.URL.Host)
	}

	_, err, _ = resolver.DownloadResolvedCoord(*lc)
	if err != nil {
		t.Fatalf("Failed to download resolved coord: %v", err)
	}

}

type mockBazurtoGrpcServer struct {
	pb.UnimplementedBazurtoServer
	server            *grpc.Server
	ResolveCoordFunc  func(context.Context, *pb.ResolveCoordRequest) (*pb.ResolveCoordResponse, error)
	DownloadCoordFunc func(req *pb.DownloadCoordRequest, srv grpc.ServerStreamingServer[pb.BinaryChunk])
}

func (o *mockBazurtoGrpcServer) ResolveCoord(ctx context.Context, rcr *pb.ResolveCoordRequest) (*pb.ResolveCoordResponse, error) {
	return o.ResolveCoordFunc(ctx, rcr)
}

func (o *mockBazurtoGrpcServer) DownloadCoord(req *pb.DownloadCoordRequest, srv grpc.ServerStreamingServer[pb.BinaryChunk]) error {
	o.DownloadCoordFunc(req, srv)
	req.Coord = "http://bazurto/owner/repo#1.2.3"
	chunkSize := 1024
	totalSize := 10 * 1024
	sent := 0
	for sent < totalSize {
		toSend := chunkSize
		if sent+toSend > totalSize {
			toSend = totalSize - sent
		}
		chunk := &pb.BinaryChunk{
			Data: make([]byte, toSend),
		}
		for i := range chunk.Data {
			chunk.Data[i] = byte((sent + i) % 256)
		}
		if err := srv.Send(chunk); err != nil {
			return err
		}
		sent += toSend
		time.Sleep(10 * time.Millisecond) // Simulate network delay
	}
	return nil
}

func (o *mockBazurtoGrpcServer) Stop() {
	o.server.Stop()
}

func startMockBazurtoGrpcServer(port int) *mockBazurtoGrpcServer {
	server := &mockBazurtoGrpcServer{}
	lis, err := net.Listen("tcp", "localhost:"+fmt.Sprintf("%d", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterBazurtoServer(s, server)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	wg.Wait()
	server.server = s
	return server
}
