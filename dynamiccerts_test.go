package grpcexperiments

import (
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/lstoll/grpcexperiments/testproto"
	"golang.org/x/net/context"
)

type kvstore struct {
	data map[string][]byte
}

func (k *kvstore) Get(key string) ([]byte, error) {
	if ret, ok := k.data[key]; ok {
		return ret, nil
	}
	return nil, fmt.Errorf("Item not found: %q", key)
}

func (k *kvstore) Put(key string, data []byte) error {
	k.data[key] = data
	return nil
}

func (k *kvstore) Delete(key string) error {
	delete(k.data, key)
	return nil
}

func TestDynamicCerts(t *testing.T) {
	store := &kvstore{data: map[string][]byte{}}

	address := "127.0.0.1:15611"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer(grpc.Creds(NewServerDynamicCertTransportCredentials(store, address, time.Now().AddDate(0, 0, 1))))
	testproto.RegisterTestProtoServer(s, &tpserver{num: "1"})
	t.Log("Starting server")
	go s.Serve(lis)

	// Let the server gen certs, start etc.
	time.Sleep(100 * time.Millisecond)

	t.Log("Starting client with cert in place")
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(NewClientDynamicCertTransportCredentials(store)))
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}
	c := testproto.NewTestProtoClient(conn)

	_, err = c.GetLBInfo(context.Background(), &testproto.LBInfoRequest{})
	if err != nil {
		t.Fatalf("Error calling RPC: %v", err)
	}

	/* TODO - need to disable retries or something on this. check for state TransientFailure/Broadcast?
	t.Log("Starting client with cert deleted from kvstore")
		store.Delete(address)
		conn, err = grpc.Dial(address, grpc.WithTransportCredentials(NewClientDynamicCertTransportCredentials(store)))
		if err != nil {
			t.Fatalf("Error connecting to server: %v", err)
		}
		c = testproto.NewTestProtoClient(conn)

		_, err = c.GetLBInfo(context.Background(), &testproto.LBInfoRequest{})
		if err != nil {
			t.Fatalf("Error calling RPC: %v", err)
		}*/
}
