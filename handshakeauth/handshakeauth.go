package handshakeauth

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"

	"golang.org/x/net/context"

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
}

type shakeresp struct {
	Successful bool
	Errormsg   string
}

// ClientHandshakeValidator is the function called when a client tries to
// connect. It receives the k/v info from the client, and should return an object to
// attach to the AuthInfo, or error if the connection is rejected
type ClientHandshakeValidator func(request map[string]interface{}) (authMetadata interface{}, err error)

func NewClientTransportCredentials(send map[string]interface{}) credentials.TransportCredentials {
	return &shaker{
		send: send,
	}
}

func NewClientTransportCredentialsWrapping(send map[string]interface{}, wrap credentials.TransportCredentials) credentials.TransportCredentials {
	return &shaker{
		send: send,
		wrap: wrap,
	}
}

func NewServerTransportCredentials(validator ClientHandshakeValidator) credentials.TransportCredentials {
	return &shaker{
		server:    true,
		validator: validator,
	}
}

func NewServerTransportCredentialsWrapping(validator ClientHandshakeValidator, wrap credentials.TransportCredentials) credentials.TransportCredentials {
	return &shaker{
		wrap:      wrap,
		server:    true,
		validator: validator,
	}
}

func (i *shaker) ClientHandshake(addr string, rawConn net.Conn, timeout time.Duration) (conn net.Conn, ai credentials.AuthInfo, err error) {
	if i.server {
		panic("This handshaker is only for client use")
	}
	// Call the wrapped handshake
	if i.wrap != nil {
		conn, ai, err = i.wrap.ClientHandshake(addr, rawConn, timeout)
		if err != nil {
			return nil, nil, err
		}
	} else {
		conn = rawConn
	}

	// Write the initiation to the server
	data, err := json.Marshal(i.send)
	if err != nil {
		conn.Close()
		return
	}
	if err = write(conn, data); err != nil {
		conn.Close()
		return
	}

	// Read the response back
	resp := &shakeresp{}
	data, err = read(conn)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, resp); err != nil {
		return
	}
	if !resp.Successful {
		conn.Close()
		err = errors.New(resp.Errormsg)
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
			return nil, nil, err
		}
	} else {
		conn = rawConn
	}

	// Read the initiation from the client
	req := &map[string]interface{}{}
	data, err := read(conn)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, req); err != nil {
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
		conn.Close()
		return
	}
	if err = write(conn, data); err != nil {
		conn.Close()
		return
	}
	if !resp.Successful {
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
