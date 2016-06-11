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

type tpserver struct {
	num string
}

func (t *tpserver) GetLBInfo(ctx context.Context, req *testproto.LBInfoRequest) (*testproto.LBInfoResponse, error) {
	return &testproto.LBInfoResponse{
		Name: t.num,
	}, nil
}

func TestEndToEnd(t *testing.T) {
	// Start some servers.
	portbase := "1560"
	servNums := []string{"1", "2", "3"}
	servers := []*grpc.Server{}
	listeners := []net.Listener{}
	t.Log("Setting up servers")
	for _, n := range servNums {
		lis, err := net.Listen("tcp", "127.0.0.1:"+portbase+n)
		if err != nil {
			panic(err)
		}
		listeners = append(listeners, lis)

		s := grpc.NewServer()
		testproto.RegisterTestProtoServer(s, &tpserver{num: n})
		servers = append(servers, s)
	}

	t.Log("Starting first 2 servers")
	go servers[0].Serve(listeners[0])
	go servers[1].Serve(listeners[1])

	targets := []string{}
	lookup := func(key string) ([]string, error) {
		if key == "testtarget" {
			return targets, nil
		}
		return nil, fmt.Errorf("Unknown target: %q", key)
	}

	t.Log("Starting client with no servers")
	conn, err := grpc.Dial("testtarget",
		grpc.WithBalancer(grpc.RoundRobin(NewPollingResolver("testtarget", 1*time.Millisecond, lookup))),
		grpc.WithInsecure(),
		grpc.WithTimeout(1*time.Second))
	if err == nil {
		t.Fatal("Did not fail to connect with no servers")
	}

	for _, n := range servNums {
		targets = append(targets, "127.0.0.1:"+portbase+n)
	}
	t.Log("Setting 2 targets and starting balancer")
	targets = []string{"127.0.0.1:" + portbase + "1", "127.0.0.1:" + portbase + "2"}
	conn, err = grpc.Dial("testtarget",
		grpc.WithBalancer(grpc.RoundRobin(NewPollingResolver("testtarget", 1*time.Millisecond, lookup))),
		grpc.WithInsecure(),
		grpc.WithTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("Error while dialing with a server list: %q", err)
	}
	defer conn.Close()

	c := testproto.NewTestProtoClient(conn)

	_, err = c.GetLBInfo(context.Background(), &testproto.LBInfoRequest{})
	if err != nil {
		t.Fatalf("Error calling RPC: %q", err)
	}

	assertSeenInReqs(t, c, 4, []string{"1", "2"})

	t.Log("Adding 3rd target to balancer, but not starting it")
	targets = append(targets, "127.0.0.1:"+portbase+"3")
	time.Sleep(2 * time.Millisecond)
	// TODO - why do we just hang here forever? Seems more a grpc problem
	/*assertSeenInReqs(t, c, 6, []string{"1", "2", "3"})*/

	t.Log("Starting 3rd server")

	go servers[2].Serve(listeners[2])
	time.Sleep(50 * time.Millisecond)
	assertSeenInReqs(t, c, 6, []string{"1", "2", "3"})

	t.Log("Stopping first server but leaving in targets")
	servers[0].Stop()
	time.Sleep(2 * time.Millisecond)
	assertSeenInReqs(t, c, 4, []string{"2", "3"})

	t.Log("Removing first server from targets")
	targets = []string{"127.0.0.1:" + portbase + "2", "127.0.0.1:" + portbase + "3"}
	time.Sleep(2 * time.Millisecond)
	assertSeenInReqs(t, c, 4, []string{"2", "3"})

	t.Log("Stopping second server and removing from targets")
	servers[1].Stop()
	targets = []string{"127.0.0.1:" + portbase + "3"}
	time.Sleep(2 * time.Millisecond)
	assertSeenInReqs(t, c, 2, []string{"3"})

	t.Log("Finished, stopping last server")
	servers[2].Stop()
}

func assertSeenInReqs(t *testing.T, c testproto.TestProtoClient, numTry int, expect []string) {
	seen := map[string]struct{}{}
	for i := 0; i < numTry; i++ {
		resp, err := c.GetLBInfo(context.Background(), &testproto.LBInfoRequest{})
		if err != nil {
			t.Fatalf("Error getting LB info: %q", err)
		}
		seen[resp.Name] = struct{}{}
	}
	for _, e := range expect {
		if _, ok := seen[e]; !ok {
			t.Fatalf("Expected %q in requests, but wasn't seen", e)
		}
	}
	for s := range seen {
		found := false
		for _, e := range expect {
			if e == s {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Found %q in requests, but wasn't expected", s)
		}
	}
}
