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
	server *grpc.Server
	conn   *grpc.ClientConn

	lis      *memlistener.MemoryListener
	closed   bool
	closedMx sync.Mutex
}

// New returns a new GRPCTestServer, configured to listen on a random local port
func New() *Server {
	return &Server{
		server: grpc.NewServer(),
		lis:    memlistener.NewMemoryListener(),
	}
}

// Start runs the server.
func (s *Server) Start() error {
	go func() {
		defer func() { s.closedMx.Unlock() }()
		s.closedMx.Lock()
		if err := s.server.Serve(s.lis); err != nil && !s.closed {
			log.Printf("Error starting test GRPC server: %s", err)
		}
	}()
	conn, err := grpc.Dial("",
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return s.lis.Dial("", "")
		}),
		grpc.WithInsecure(),
	)
	s.conn = conn
	return err
}

// Server returns the gRPC server in play. Services can be registered on this.
func (s *Server) Server() *grpc.Server {
	return s.server
}

// Client returns the gRPC client connection.
func (s *Server) Client() *grpc.ClientConn {
	return s.conn
}

// Close closes the client connection, and stops the server from listening
func (t *Server) Close() error {
	if err := t.conn.Close(); err != nil {
		return err
	}
	t.server.Stop()
	defer func() { t.closedMx.Unlock() }()
	t.closedMx.Lock()
	t.closed = true
	return nil
}
