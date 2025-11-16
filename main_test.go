package main

import (
	"context"
	"log"
	"testing"
	"time"

	pb "github.com/bazurto/bz/grpc/bazurto" // Adjust the import path as necessary
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestGrpcServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		metadata.NewOutgoingContext(context.Background(),
			metadata.New(map[string]string{"xtoken": "12345"}),
		), time.Second*5,
	)
	defer cancel()

	clientConn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer clientConn.Close()

	client := pb.NewBazurtoClient(clientConn)

	resp, err := client.GetLatestVersion(ctx, &pb.LatestVersionRequest{
		Owner: "bazurto",
		Repo:  "bz",
	})
	if err != nil {
		log.Fatalf("Error calling GetLatestVersion: %v", err)
	}

	log.Printf("Latest version: %s", resp.Version)
}
