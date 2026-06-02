package grpc

import (
	"context"
	"time"

	pb "github.com/aegis/firewall/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AIClient wraps the gRPC connection to the Python AI Engine.
type AIClient struct {
	client pb.AIAnalyzerClient
	conn   *grpc.ClientConn
}

// NewAIClient establishes a connection to the Python gRPC server.
func NewAIClient(target string) (*AIClient, error) {
	// For this MVP, we use plaintext connections on the internal network.
	// In production, this would use TLS certificates.
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewAIAnalyzerClient(conn)
	return &AIClient{client: client, conn: conn}, nil
}

// Close gracefully shuts down the gRPC connection.
func (c *AIClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// AnalyzeRequest sends the request metadata to the AI engine to get a security verdict.
func (c *AIClient) AnalyzeRequest(ctx context.Context, req *pb.AnalyzeRequestMessage) (*pb.AnalyzeResponseMessage, error) {
	// Crucial Security Design: 2-second timeout!
	// If the AI Engine goes down or hangs, the Go proxy will abort the gRPC call
	// after 2 seconds to prevent the entire API gateway from freezing.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return c.client.AnalyzeRequest(ctx, req)
}
