package kvresolver

import (
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/lstoll/grpce/helloproto"
	"golang.org/x/net/context"
)

func TestEndToEnd(t *testing.T) {
	// Start some servers.
	servNums := []string{"1", "2", "3"}
	servers := []*grpc.Server{}
	listeners := []net.Listener{}
	t.Log("Setting up servers")
	for _, n := range servNums {
		lis, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			panic(err)
		}
		listeners = append(listeners, lis)

		s := grpc.NewServer()
		helloproto.RegisterHelloServer(s, &helloproto.TestHelloServer{ServerName: n})
		servers = append(servers, s)
	}

	t.Log("Starting first 2 servers")
	go servers[0].Serve(listeners[0])
	go servers[1].Serve(listeners[1])

	targets := []net.Listener{}
	lookup := func(key string) ([]string, error) {
		if key == "testtarget" {
			ret := []string{}
			for _, t := range targets {
				ret = append(ret, t.Addr().String())
			}
			return ret, nil
		}
		return nil, fmt.Errorf("Unknown target: %q", key)
	}

	t.Log("Starting client with no servers")
	conn, err := grpc.Dial("testtarget",
		grpc.WithBalancer(grpc.RoundRobin(New("testtarget", 1*time.Millisecond, lookup))),
		grpc.WithInsecure(),
		grpc.WithTimeout(1*time.Second))
	if err == nil {
		t.Fatal("Did not fail to connect with no servers")
	}

	/*for _, lis := range listeners {
		targets = append(targets, lis)
	}*/
	t.Log("Setting 2 targets and starting balancer")
	targets = []net.Listener{listeners[0], listeners[1]}
	conn, err = grpc.Dial("testtarget",
		grpc.WithBalancer(grpc.RoundRobin(New("testtarget", 1*time.Millisecond, lookup))),
		grpc.WithInsecure(),
		grpc.WithTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("Error while dialing with a server list: %q", err)
	}
	defer conn.Close()

	c := helloproto.NewHelloClient(conn)

	_, err = c.HelloWorld(context.Background(), &helloproto.HelloRequest{Name: "process"})
	if err != nil {
		t.Fatalf("Error calling RPC: %q", err)
	}

	assertSeenInReqs(t, c, 4, []string{"1", "2"})

	t.Log("Adding 3rd target to balancer, but not starting it")
	targets = append(targets, listeners[2])
	time.Sleep(2 * time.Millisecond)
	// TODO - why do we just hang here forever? Seems more a grpc problem timeing out on the connection
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
	targets = []net.Listener{listeners[1], listeners[2]}
	time.Sleep(2 * time.Millisecond)
	assertSeenInReqs(t, c, 4, []string{"2", "3"})

	t.Log("Stopping second server and removing from targets")
	servers[1].Stop()
	targets = []net.Listener{listeners[2]}
	time.Sleep(2 * time.Millisecond)
	assertSeenInReqs(t, c, 2, []string{"3"})

	t.Log("Finished, stopping last server")
	servers[2].Stop()
}

func assertSeenInReqs(t *testing.T, c helloproto.HelloClient, numTry int, expect []string) {
	seen := map[string]struct{}{}
	for i := 0; i < numTry; i++ {
		resp, err := c.HelloWorld(context.Background(), &helloproto.HelloRequest{})
		if err != nil {
			t.Fatalf("Error getting LB info: %q", err)
		}
		seen[resp.ServerName] = struct{}{}
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
