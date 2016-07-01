package gometrics

import (
	"net"
	"testing"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/lstoll/grpce/helloproto"
	"github.com/rcrowley/go-metrics"
)

type sc struct {
	count int64
}

func TestMetricsEnd2End(t *testing.T) {
	registry := metrics.NewRegistry()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	s := grpc.NewServer(
		grpc.StreamInterceptor(NewStreamServerInterceptor(registry, "p")),
		grpc.UnaryInterceptor(NewUnaryServerInterceptor(registry, "p")),
	)
	helloproto.RegisterHelloServer(s, &helloproto.TestHelloServer{ServerName: "testserver"})
	go func() { _ = s.Serve(lis) }()
	defer s.Stop()
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure(), grpc.WithTimeout(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	c := helloproto.NewHelloClient(conn)
	_, _ = c.HelloWorld(context.Background(), &helloproto.HelloRequest{Name: "instrument"})
	_, _ = c.HelloWorld(context.Background(), &helloproto.HelloRequest{Name: "instrument"})
	_, _ = c.HelloWorld(context.Background(), &helloproto.HelloRequest{Name: "instrument"})
	if count := registry.Get("p.grpc.server.msgs_sent.unary.helloproto.Hello.HelloWorld").(metrics.Counter).Count(); count != 3 {
		t.Errorf("Expected 3 messages sent, got %d", count)
	}
	if count := registry.Get("p.grpc.server.handled.unary.helloproto.Hello.HelloWorld.OK").(metrics.Counter).Count(); count != 3 {
		t.Errorf("Expected 3 OK responses, got %d", count)
	}
}
