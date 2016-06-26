package handshakeauth

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"time"

	"google.golang.org/grpc/credentials"
)

type HandshakeAuthInfo struct{}

func (h *HandshakeAuthInfo) AuthType() string {
	return "HandshakeAuthInfo"
}

type instanceauthtransport struct {
	wrap credentials.TransportCredentials
}

type handshake struct {
	Doc []byte
	Sig []byte
}

func NewTransportCredentials() credentials.TransportCredentials {
	return &instanceauthtransport{}
}

func NewTransportCredentialsWraping(wrap credentials.TransportCredentials) credentials.TransportCredentials {
	return &instanceauthtransport{
		wrap: wrap,
	}
}

func (i *instanceauthtransport) ClientHandshake(addr string, rawConn net.Conn, timeout time.Duration) (conn net.Conn, ai credentials.AuthInfo, err error) {
	// Call the wrapped handshake
	if i.wrap != nil {
		conn, ai, err = i.wrap.ClientHandshake(addr, rawConn, timeout)
		if err != nil {
			return nil, nil, err
		}
	} else {
		conn = rawConn
	}

	// Send the doc down the conn.
	handshake := &handshake{}
	data, err := json.Marshal(handshake)
	if err != nil {
		conn.Close()
		return
	}
	hdr := make([]byte, 4)
	binary.LittleEndian.PutUint32(hdr, uint32(len(data)))
	_, err = conn.Write(hdr)
	if err != nil {
		conn.Close()
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		conn.Close()
		return
	}

	// Just carry on. Server will close connection if creds are invalid
	// TODO - consider some kind of reply for better logging?

	return
}

func (i *instanceauthtransport) ServerHandshake(rawConn net.Conn) (conn net.Conn, ai credentials.AuthInfo, err error) {
	// Run the wrapped server handshake
	if i.wrap != nil {
		conn, _, err = i.wrap.ServerHandshake(rawConn)
		if err != nil {
			return nil, nil, err
		}
	} else {
		conn = rawConn
	}

	// Read for out message and doc
	hdr := make([]byte, 4)
	_, err = io.ReadAtLeast(conn, hdr, 4)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	toRead := binary.LittleEndian.Uint32(hdr)
	data := make([]byte, toRead)
	_, err = io.ReadAtLeast(conn, data, int(toRead))
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	handshake := &handshake{}
	err = json.Unmarshal(data, handshake)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	// validate it. If it's not valid, return an error

	ai = &HandshakeAuthInfo{}
	/*err = json.Unmarshal(handshake.Doc, ai)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}*/

	// pass off the connection
	return conn, ai, err
}

func (i *instanceauthtransport) Info() credentials.ProtocolInfo {
	return i.wrap.Info()
}

// InstanceIdentityDocumentAuthInfoFromContext will take the requests's context and return the InstanceIdentityDocumentAuthInfo for the caller, or error if it can't look this information up
/*func InstanceIdentityDocumentAuthInfoFromContext(ctx context.Context) (*InstanceIdentityDocumentAuthInfo, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from ctx")
	}
	ia, ok := pr.AuthInfo.(*InstanceIdentityDocumentAuthInfo)
	if !ok {
		return nil, fmt.Errorf("failed to get InstanceIdentityDocumentAuthInfo from peer")
	}
	return ia, nil
}*/
