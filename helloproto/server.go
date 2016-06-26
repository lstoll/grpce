package helloproto

import (
	"fmt"

	"golang.org/x/net/context"
)

type TestHelloServer struct {
	ServerName string
}

func (h *TestHelloServer) HelloWorld(ctx context.Context, req *HelloRequest) (*HelloResponse, error) {
	return &HelloResponse{
		Message:    fmt.Sprintf("Hello, %s!", req.Name),
		ServerName: h.ServerName,
	}, nil
}
