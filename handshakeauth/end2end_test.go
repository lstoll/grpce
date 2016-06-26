package handshakeauth

import (
	"net"
	"testing"
	"time"

	"github.com/lstoll/grpce/helloproto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestHandshakeAuth(t *testing.T) {
	// Instance metadata server. Start with valid docs

	// Start a server
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer(grpc.Creds(
		NewTransportCredentials(),
	))
	helloproto.RegisterHelloServer(s, &helloproto.TestHelloServer{})
	t.Log("Starting server")
	go s.Serve(lis)

	// Let the server gen certs, start etc.
	time.Sleep(100 * time.Millisecond)

	// Should work out of the box
	t.Log("Starting client with cert in place")
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(
		NewTransportCredentials(),
	))
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}
	c := helloproto.NewHelloClient(conn)

	_, err = c.HelloWorld(context.Background(), &helloproto.HelloRequest{Name: "Handshakin"})
	if err != nil {
		t.Fatalf("Error calling RPC: %v", err)
	}
	/*if resp.Name != "i-0e90d494ecf1ea4bc" {
		t.Fatalf("Server didn't return back our instance ID")
	}*/

	// TODO - again, failing test. When we can stop the retry behavior

	s.Stop()
}
