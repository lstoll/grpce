package h2c

import (
	"net"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"

	"github.com/lstoll/grpce/helloproto"
)

type helloH2C struct{}

func (helloH2C) HelloWorld(ctx context.Context, _ *helloproto.HelloRequest) (*helloproto.HelloResponse, error) {
	return &helloproto.HelloResponse{
		Message:    "Hello over h2c!",
		ServerName: "h2c",
	}, nil
}

func TestEnd2End(t *testing.T) {
	http2.DebugGoroutines = true

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	s := grpc.NewServer()
	helloproto.RegisterHelloServer(s, helloH2C{})

	srv := &Server{
		HTTP2Handler:      s,
		NonUpgradeHandler: http.HandlerFunc(http.NotFound),
	}

	go http.Serve(ln, srv)

	for {
		if conn, err := net.Dial("tcp", ln.Addr().String()); err == nil {
			conn.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	conn, err := grpc.Dial(ln.Addr().String(), grpc.WithDialer(Dialer{}.DialGRPC), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}

	c := helloproto.NewHelloClient(conn)

	res, err := c.HelloWorld(context.Background(), new(helloproto.HelloRequest))
	if err != nil {
		t.Error(err)
	}
	if want, got := "Hello over h2c!", res.Message; want != got {
		t.Errorf("want response message %q, got %q", want, got)
	}
	if want, got := "h2c", res.ServerName; want != got {
		t.Errorf("want server name %q, got %q", want, got)
	}
}
