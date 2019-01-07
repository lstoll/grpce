// Package inproc implements a simple gRPC server & client that communicate in
// process This is useful for testing, or for services that don't need to cross
// process bounds.
package inproc

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/hydrogen18/memlistener"
	"google.golang.org/grpc"
)

// Server is an in-process gRPC server.
type Server struct {
	// Server points to the gRPC server instances. Services should be registered on this
	Server *grpc.Server
	// ClientConn is a handle to a connection to the server. This can be used to
	// get clients.
	ClientConn *grpc.ClientConn

	lis      *memlistener.MemoryListener
	closed   bool
	closedMx sync.Mutex
}

// New returns a new Server.
func New() *Server {
	return &Server{
		Server: grpc.NewServer(),
		lis:    memlistener.NewMemoryListener(),
	}
}

// Start runs the server. This call starts the server in a routine, and returns
// immediately.
func (s *Server) Start() error {
	go func() {
		defer func() { s.closedMx.Unlock() }()
		s.closedMx.Lock()
		if err := s.Server.Serve(s.lis); err != nil && !s.closed {
			log.Printf("Error starting test GRPC server: %s", err)
		}
	}()
	conn, err := grpc.Dial("",
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return s.lis.Dial("", "")
		}),
		grpc.WithInsecure(),
	)
	s.ClientConn = conn
	return err
}

// Close closes the client connection, and stops the server from listening.
func (s *Server) Close() error {
	if err := s.ClientConn.Close(); err != nil {
		return err
	}
	s.Server.Stop()
	defer func() { s.closedMx.Unlock() }()
	s.closedMx.Lock()
	s.closed = true
	return nil
}
