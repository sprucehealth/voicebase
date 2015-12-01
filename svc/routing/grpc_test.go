package routing

import (
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type routingError struct {
	Code    string
	Message string
}

func (r *routingError) Error() string {
	return fmt.Sprintf("Code: %s, Message: %s", r.Code, r.Message)
}

type routingServer struct{}

func (r *routingServer) RouteMessage(context.Context, *RouteMessageRequest) (*RouteMessageResponse, error) {
	return &RouteMessageResponse{}, nil
}

func TestGRPC(t *testing.T) {

	// setup server
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		s := grpc.NewServer()
		RegisterRoutingServer(s, &routingServer{})
		s.Serve(lis)
	}()

	// test client
	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithTimeout(1*time.Second))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := NewRoutingClient(conn)
	if _, err := c.RouteMessage(context.Background(), &RouteMessageRequest{}); err != nil {
		t.Fatal(err)
	}
}
