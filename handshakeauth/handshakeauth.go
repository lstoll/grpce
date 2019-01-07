package handshakeauth

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"

	"github.com/lstoll/grpce/reporters"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

type handshakeAuthInfo struct {
	clientMetadata  interface{}
	wrappedAuthInfo credentials.AuthInfo
}

func (h *handshakeAuthInfo) AuthType() string {
	return "HandshakeAuthInfo"
}

type shaker struct {
	server    bool
	wrap      credentials.TransportCredentials
	send      map[string]interface{}
	validator ClientHandshakeValidator
	opts      *hsOpts
}

type shakeresp struct {
	Successful bool
	Errormsg   string
}

type hsOpts struct {
	errorReporter   reporters.ErrorReporter
	metricsReporter reporters.MetricsReporter
}

type HSOption func(*hsOpts)

func WithErrorReporter(er reporters.ErrorReporter) HSOption {
	return func(o *hsOpts) {
		o.errorReporter = er
	}
}

func WithMetricsReporter(mr reporters.MetricsReporter) HSOption {
	return func(o *hsOpts) {
		o.metricsReporter = mr
	}
}

// ClientHandshakeValidator is the function called when a client tries to
// connect. It receives the k/v info from the client, and should return an object to
// attach to the AuthInfo, or error if the connection is rejected
type ClientHandshakeValidator func(request map[string]interface{}) (authMetadata interface{}, err error)

func NewClientTransportCredentials(send map[string]interface{}, opts ...HSOption) credentials.TransportCredentials {
	hso := &hsOpts{}
	for _, opt := range opts {
		opt(hso)
	}
	return &shaker{
		send: send,
		opts: hso,
	}
}

func NewClientTransportCredentialsWrapping(send map[string]interface{}, wrap credentials.TransportCredentials, opts ...HSOption) credentials.TransportCredentials {
	hso := &hsOpts{}
	for _, opt := range opts {
		opt(hso)
	}
	return &shaker{
		send: send,
		wrap: wrap,
		opts: hso,
	}
}

func NewServerTransportCredentials(validator ClientHandshakeValidator, opts ...HSOption) credentials.TransportCredentials {
	hso := &hsOpts{}
	for _, opt := range opts {
		opt(hso)
	}
	return &shaker{
		server:    true,
		validator: validator,
		opts:      hso,
	}
}

func NewServerTransportCredentialsWrapping(validator ClientHandshakeValidator, wrap credentials.TransportCredentials, opts ...HSOption) credentials.TransportCredentials {
	hso := &hsOpts{}
	for _, opt := range opts {
		opt(hso)
	}
	return &shaker{
		wrap:      wrap,
		server:    true,
		validator: validator,
		opts:      hso,
	}
}

// Clone returns a copy of the credentials
func (i *shaker) Clone() credentials.TransportCredentials {
	var wrapped credentials.TransportCredentials
	if i.wrap != nil {
		wrapped = i.wrap.Clone()
	}
	return &shaker{
		wrap: wrapped,
		// We can re-use these
		server:    i.server,
		validator: i.validator,
		opts:      i.opts,
	}
}

// OverrideServerName overrides the server name, used before dial
func (i *shaker) OverrideServerName(serverNameOverride string) error {
	if i.wrap != nil {
		i.wrap.OverrideServerName(serverNameOverride)
	}
	return nil
}

func (i *shaker) ClientHandshake(ctx context.Context, addr string, rawConn net.Conn) (conn net.Conn, ai credentials.AuthInfo, err error) {
	if i.server {
		panic("This handshaker is only for client use")
	}
	// Call the wrapped handshake
	if i.wrap != nil {
		conn, ai, err = i.wrap.ClientHandshake(ctx, addr, rawConn)
		if err != nil {
			reporters.ReportError(i.opts.errorReporter, err)
			reporters.ReportCount(i.opts.metricsReporter, "handshakeauth.client.wrapped.errors", 1)
			return nil, nil, err
		}
	} else {
		conn = rawConn
	}

	// Write the initiation to the server
	data, err := json.Marshal(i.send)
	if err != nil {
		reporters.ReportError(i.opts.errorReporter, err)
		conn.Close()
		return
	}
	if err = write(conn, data); err != nil {
		reporters.ReportError(i.opts.errorReporter, err)
		conn.Close()
		return
	}

	// Read the response back
	resp := &shakeresp{}
	data, err = read(conn)
	if err != nil {
		reporters.ReportError(i.opts.errorReporter, err)
		return
	}
	if err = json.Unmarshal(data, resp); err != nil {
		reporters.ReportError(i.opts.errorReporter, err)
		return
	}
	if !resp.Successful {
		conn.Close()
		err = errors.New(resp.Errormsg)
		reporters.ReportError(i.opts.errorReporter, err)
		reporters.ReportCount(i.opts.metricsReporter, "handshakeauth.client.unsuccessful", 1)
		return
	}

	return
}

func (i *shaker) ServerHandshake(rawConn net.Conn) (conn net.Conn, ai credentials.AuthInfo, err error) {
	if !i.server {
		panic("This handshaker is only for server use")
	}
	// Run the wrapped server handshake
	if i.wrap != nil {
		conn, ai, err = i.wrap.ServerHandshake(rawConn)
		if err != nil {
			reporters.ReportError(i.opts.errorReporter, err)
			reporters.ReportCount(i.opts.metricsReporter, "handshakeauth.server.wrapped.errors", 1)
			return nil, nil, err
		}
	} else {
		conn = rawConn
	}

	// Read the initiation from the client
	req := &map[string]interface{}{}
	data, err := read(conn)
	if err != nil {
		reporters.ReportError(i.opts.errorReporter, err)
		return
	}
	if err = json.Unmarshal(data, req); err != nil {
		reporters.ReportError(i.opts.errorReporter, err)
		return
	}
	// Call the validator
	resp := &shakeresp{}
	md, err := i.validator(*req)
	if err != nil {
		resp.Successful = false
		resp.Errormsg = err.Error()
	} else {
		resp.Successful = true
	}
	data, err = json.Marshal(resp)
	if err != nil {
		reporters.ReportError(i.opts.errorReporter, err)
		conn.Close()
		return
	}
	if err = write(conn, data); err != nil {
		reporters.ReportError(i.opts.errorReporter, err)
		conn.Close()
		return
	}
	if !resp.Successful {
		reporters.ReportError(i.opts.errorReporter, err)
		conn.Close()
		return
	}

	ai = &handshakeAuthInfo{
		clientMetadata:  md,
		wrappedAuthInfo: ai,
	}

	return
}

func (i *shaker) Info() credentials.ProtocolInfo {
	if i.wrap != nil {
		return i.wrap.Info()
	}
	return credentials.ProtocolInfo{} // TODO - do we have info?
}

func read(r io.Reader) ([]byte, error) {
	hdr := make([]byte, 4)
	_, err := io.ReadAtLeast(r, hdr, 4)
	if err != nil {
		return nil, err
	}
	toRead := binary.LittleEndian.Uint32(hdr)
	data := make([]byte, toRead)
	_, err = io.ReadAtLeast(r, data, int(toRead))
	return data, err
}

func write(w io.Writer, d []byte) error {
	hdr := make([]byte, 4)
	binary.LittleEndian.PutUint32(hdr, uint32(len(d)))
	_, err := w.Write(hdr)
	if err != nil {
		return err
	}
	_, err = w.Write(d)
	if err != nil {
		return err
	}
	return nil
}

func ClientMetadataFromContext(ctx context.Context) interface{} {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return nil
	}
	ia, ok := pr.AuthInfo.(*handshakeAuthInfo)
	if !ok {
		return nil
	}
	return ia.clientMetadata
}
