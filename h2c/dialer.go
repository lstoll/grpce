package h2c

import (
	"bufio"
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Dialer connects to a HTTP 1.1 server and performs an h2c upgrade to an HTTP2 connection.
type Dialer struct {
	Dialer *net.Dialer
	URL    *url.URL
}

// DialContext connects to the address on the named network using the provided context.
func (d Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	dialfn := http.DefaultTransport.(*http.Transport).DialContext
	if d.Dialer != nil && d.Dialer.DialContext != nil {
		dialfn = d.Dialer.DialContext
	}

	conn, err := dialfn(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	u := "http://" + addr
	if d.URL != nil {
		u = d.URL.String()
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")

	if err := req.Write(conn); err != nil {
		return nil, err
	}

	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusSwitchingProtocols {
		return nil, errors.New("h2c upgrade failed, recieved non 101 response")
	}
	if strings.ToLower(res.Header.Get("Connection")) != "upgrade" {
		return nil, errors.New("h2c upgrade failed, bad Connection header in response")
	}
	if strings.ToLower(res.Header.Get("Upgrade")) != "h2c" {
		return nil, errors.New("h2c upgrade failed, bad Upgrade header in response")
	}
	if buf, err := ioutil.ReadAll(res.Body); len(buf) > 0 || err != nil {
		return nil, errors.New("h2c upgrade failed, upgrade response body was non empty")
	}

	return conn, nil
}

// Dial connects to the address on the named network.
func (d Dialer) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

// DialGRPC connects to the address before timeout.
func (d Dialer) DialGRPC(addr string, timeout time.Duration) (net.Conn, error) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return d.DialContext(ctx, "tcp", addr)
}