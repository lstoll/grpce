package grpcexperiments

import (
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/fullsailor/pkcs7"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

var (
	identityDocURL = "http://169.254.169.254/latest/dynamic/instance-identity/document"
	pkcs7URL       = "http://169.254.169.254/latest/dynamic/instance-identity/pkcs7"
)

var awsPubKey = `-----BEGIN CERTIFICATE-----
MIIC7TCCAq0CCQCWukjZ5V4aZzAJBgcqhkjOOAQDMFwxCzAJBgNVBAYTAlVTMRkw
FwYDVQQIExBXYXNoaW5ndG9uIFN0YXRlMRAwDgYDVQQHEwdTZWF0dGxlMSAwHgYD
VQQKExdBbWF6b24gV2ViIFNlcnZpY2VzIExMQzAeFw0xMjAxMDUxMjU2MTJaFw0z
ODAxMDUxMjU2MTJaMFwxCzAJBgNVBAYTAlVTMRkwFwYDVQQIExBXYXNoaW5ndG9u
IFN0YXRlMRAwDgYDVQQHEwdTZWF0dGxlMSAwHgYDVQQKExdBbWF6b24gV2ViIFNl
cnZpY2VzIExMQzCCAbcwggEsBgcqhkjOOAQBMIIBHwKBgQCjkvcS2bb1VQ4yt/5e
ih5OO6kK/n1Lzllr7D8ZwtQP8fOEpp5E2ng+D6Ud1Z1gYipr58Kj3nssSNpI6bX3
VyIQzK7wLclnd/YozqNNmgIyZecN7EglK9ITHJLP+x8FtUpt3QbyYXJdmVMegN6P
hviYt5JH/nYl4hh3Pa1HJdskgQIVALVJ3ER11+Ko4tP6nwvHwh6+ERYRAoGBAI1j
k+tkqMVHuAFcvAGKocTgsjJem6/5qomzJuKDmbJNu9Qxw3rAotXau8Qe+MBcJl/U
hhy1KHVpCGl9fueQ2s6IL0CaO/buycU1CiYQk40KNHCcHfNiZbdlx1E9rpUp7bnF
lRa2v1ntMX3caRVDdbtPEWmdxSCYsYFDk4mZrOLBA4GEAAKBgEbmeve5f8LIE/Gf
MNmP9CM5eovQOGx5ho8WqD+aTebs+k2tn92BBPqeZqpWRa5P/+jrdKml1qx4llHW
MXrs3IgIb6+hUIB+S8dz8/mmO0bpr76RoZVCXYab2CZedFut7qc3WUH9+EUAH5mw
vSeDCOUMYQR7R9LINYwouHIziqQYMAkGByqGSM44BAMDLwAwLAIUWXBlk40xTwSw
7HX32MxXYruse9ACFBNGmdX2ZBrVNGrN9N2f6ROk0k9K
-----END CERTIFICATE-----`

type InstanceIdentityDocumentAuthInfo struct {
	InstanceID       string    `json:"instanceId"`
	AccountID        string    `json:"accountId"`
	PrivateIP        string    `json:"privateIp"`
	Region           string    `json:"region"`
	AvailabilityZone string    `json:"availabilityZone"`
	PendingTime      time.Time `json:"pendingTime"`
}

func (i *InstanceIdentityDocumentAuthInfo) AuthType() string {
	return "InstanceIdentityDocument"
}

type instanceauthtransport struct {
	wrap credentials.TransportCredentials
}

type handshake struct {
	Doc []byte
	Sig []byte
}

func NewInstanceAuthTransportCredentials(wrap credentials.TransportCredentials) credentials.TransportCredentials {
	return &instanceauthtransport{
		wrap: wrap,
	}
}

func (i *instanceauthtransport) ClientHandshake(addr string, rawConn net.Conn, timeout time.Duration) (net.Conn, credentials.AuthInfo, error) {
	// Load the identity doc & sig
	doc, err := httpGET(identityDocURL, 10)
	if err != nil {
		return nil, nil, err
	}
	p7, err := httpGET(pkcs7URL, 10)
	if err != nil {
		return nil, nil, err
	}

	// Call the wrapped handshake
	conn, ai, err := i.wrap.ClientHandshake(addr, rawConn, timeout)

	// Send the doc down the conn. Format is "BEGINIDENTITYDOC\n<doc>\n<sig>\nENDIDENTITYDOC
	handshake := &handshake{
		Doc: doc,
		Sig: p7,
	}
	data, err := json.Marshal(handshake)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	hdr := make([]byte, 4)
	binary.LittleEndian.PutUint32(hdr, uint32(len(data)))
	_, err = conn.Write(hdr)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	_, err = conn.Write(data)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	// Just carry on. Server will close connection if creds are invalid
	// TODO - consider some kind of reply for better logging?

	return conn, ai, err
}

func (i *instanceauthtransport) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	// Run the wrapped server handshake
	conn, _, err := i.wrap.ServerHandshake(rawConn)
	if err != nil {
		return nil, nil, err
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
	if !verifyDocToPKCS7(handshake.Doc, handshake.Sig) {
		conn.Close()
		return nil, nil, fmt.Errorf("Connection from %q passed invalid doc %q with sig %q", conn.RemoteAddr(), handshake.Doc, handshake.Sig)
	}

	ai := &InstanceIdentityDocumentAuthInfo{}
	err = json.Unmarshal(handshake.Doc, ai)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	// pass off the connection
	return conn, ai, err
}

func (i *instanceauthtransport) Info() credentials.ProtocolInfo {
	return i.wrap.Info()
}

// InstanceIdentityDocumentAuthInfoFromContext will take the requests's context and return the InstanceIdentityDocumentAuthInfo for the caller, or error if it can't look this information up
func InstanceIdentityDocumentAuthInfoFromContext(ctx context.Context) (*InstanceIdentityDocumentAuthInfo, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from ctx")
	}
	ia, ok := pr.AuthInfo.(*InstanceIdentityDocumentAuthInfo)
	if !ok {
		return nil, fmt.Errorf("failed to get InstanceIdentityDocumentAuthInfo from peer")
	}
	return ia, nil
}

// verifyDocToPKCS7 takes a instance doc and the pkcs7 signed version
// and returns true if the signature is correct, and was signed by
// AWS's public key
func verifyDocToPKCS7(doc, pkcs7signed []byte) bool {
	sigPEM := "-----BEGIN PKCS7-----\n" + string(pkcs7signed) + "\n-----END PKCS7-----"
	sigDecode, _ := pem.Decode([]byte(sigPEM))
	if sigDecode == nil {
		return false
	}

	awsBlock, _ := pem.Decode([]byte(awsPubKey))
	if awsBlock == nil {
		panic("failed to parse AWS cert PEM")
	}
	awsCert, err := x509.ParseCertificate(awsBlock.Bytes)
	if err != nil {
		panic("failed to parse AWS cert PEM")
	}

	p7, err := pkcs7.Parse(sigDecode.Bytes)
	if err != nil {
		return false
	}
	p7.Content = doc
	p7.Certificates = []*x509.Certificate{awsCert}

	err = p7.Verify()
	if err != nil {
		return false
	}
	// if we made it here the document has a valid signature
	return true
}

// httpGET is a ghetto retrying HTTP client. Will return the fetched body
func httpGET(url string, maxRetries int) ([]byte, error) {
	retries := 0
	delay := 2 * time.Second
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(url)
		if err != nil {
			retries++
			time.Sleep(delay)
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			retries++
			time.Sleep(delay)
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("Tried to fetch %q %d times, but all failed", url, maxRetries)
}
