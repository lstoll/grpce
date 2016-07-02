package handshakeauth

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/lstoll/grpce/helloproto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type hs struct{}

func (h *hs) HelloWorld(ctx context.Context, req *helloproto.HelloRequest) (*helloproto.HelloResponse, error) {
	md := ClientMetadataFromContext(ctx)
	return &helloproto.HelloResponse{
		Message:    fmt.Sprintf("Hello, %s!", req.Name),
		ServerName: fmt.Sprintf("%s", md),
	}, nil
}

type errcatcher struct {
	err error
}

func (e *errcatcher) ReportError(err error) {
	e.err = err
}

func TestHandshakeAuth(t *testing.T) {
	// Instance metadata server. Start with valid docs

	// Start a server
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	validator := func(req map[string]interface{}) (interface{}, error) {
		keyVal, ok := req["key"]
		if !ok {
			return nil, errors.New("no key")
		}
		key, ok := keyVal.(string)
		if !ok {
			return nil, errors.New("key not string")
		}
		return key, nil
	}
	s := grpc.NewServer(grpc.Creds(
		NewServerTransportCredentials(validator),
	))
	helloproto.RegisterHelloServer(s, &hs{})
	t.Log("Starting server")
	go s.Serve(lis)

	// Let the server gen certs, start etc.
	time.Sleep(100 * time.Millisecond)

	// Should work out of the box
	t.Log("Starting client with key set")
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(
		NewClientTransportCredentials(map[string]interface{}{"key": "keygoeshere"}),
	))
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}
	c := helloproto.NewHelloClient(conn)

	resp, err := c.HelloWorld(context.Background(), &helloproto.HelloRequest{Name: "Handshakin"})
	if err != nil {
		t.Fatalf("Error calling RPC: %v", err)
	}
	if resp.ServerName != "keygoeshere" {
		t.Fatalf("Server didn't return back our key")
	}

	// TODO - this does "Fail", but the client enternally reconnects. Investigate how to reject client?
	// Alt, check underlying connection state
	t.Log("Trying client connection with no key")
	ec := &errcatcher{}
	conn, err = grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(
		NewClientTransportCredentials(map[string]interface{}{}, HSOption(WithErrorReporter(ec))),
	))
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}
	c = helloproto.NewHelloClient(conn)

	success := make(chan struct{}, 1)
	go func() {
		for {
			if ec.err != nil {
				success <- struct{}{}
				break
			}
		}
	}()
	select {
	case <-time.After(time.Second * 1):
		t.Error("Connection should have errored, but error not caught")
	case <-success:
	}

	conn.Close()

	s.Stop()
}
