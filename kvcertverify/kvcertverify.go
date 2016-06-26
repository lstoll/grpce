package grpcexperiments

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc/credentials"
)

type KVStore interface {
	Get(key string) ([]byte, error)
	Put(key string, data []byte) error
	Delete(key string) error
}

type dcerttransport struct {
	store KVStore
	// For clients
	clientcredsMu sync.Mutex
	clientcreds   map[string]credentials.TransportCredentials
	// For servers
	servercreds   credentials.TransportCredentials
	serveraddress string
	validUntil    time.Time
}

func NewClientTransportCredentials(store KVStore) credentials.TransportCredentials {
	return &dcerttransport{
		store:       store,
		clientcreds: map[string]credentials.TransportCredentials{},
	}
}

func NewServerTransportCredentials(store KVStore, address string, validUntil time.Time) credentials.TransportCredentials {
	// TODO - expiry/auto-renew?

	// Generate a certificate, save it on outselves, and put it on the KV store.
	cert, err := genX509KeyPair(address, validUntil)
	if err != nil {
		// Log or something, this is a pretty "fatal" error. Maybe even panic
		return nil
	}
	p := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	store.Put(address, p)
	servercreds := credentials.NewServerTLSFromCert(cert)

	return &dcerttransport{
		store:         store,
		serveraddress: address,
		validUntil:    validUntil,
		servercreds:   servercreds,
	}
}

func (d *dcerttransport) ClientHandshake(addr string, rawConn net.Conn, timeout time.Duration) (net.Conn, credentials.AuthInfo, error) {
	if d.clientcreds == nil {
		return nil, nil, errors.New("Credentials not initialized for client use via NewClientDynamicCertTransportCredentials")
	}
	// Be brutal and lazy
	d.clientcredsMu.Lock()
	defer d.clientcredsMu.Unlock()
	if _, ok := d.clientcreds[addr]; !ok {
		// Build client creds for this host
		rawCert, err := d.store.Get(addr)
		if err != nil {
			return nil, nil, err
		}
		capool := x509.NewCertPool()
		capool.AppendCertsFromPEM(rawCert)
		d.clientcreds[addr] = credentials.NewClientTLSFromCert(capool, addr)
	}
	return d.clientcreds[addr].ClientHandshake(addr, rawConn, timeout)
}

func (d *dcerttransport) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	if d.serveraddress == "" {
		return nil, nil, errors.New("Credentials not initialized for server use via NewServerDynamicCertTransportCredentials")
	}
	return d.servercreds.ServerHandshake(rawConn)
}

func (d *dcerttransport) Info() credentials.ProtocolInfo {
	// same as tlsCreds, we mostly just wrap it
	return credentials.ProtocolInfo{
		SecurityProtocol: "tls",
		SecurityVersion:  "1.2",
	}
}

func genX509KeyPair(name string, validUntil time.Time) (*tls.Certificate, error) {
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject: pkix.Name{
			CommonName:         name,
			Country:            []string{"internet"},
			Organization:       []string{"grpc"},
			OrganizationalUnit: []string{"dynamiccert"},
		},
		NotBefore:             now,
		NotAfter:              validUntil,
		SubjectKeyId:          []byte{113, 117, 105, 99, 107, 115, 101, 114, 118, 101},
		BasicConstraintsValid: true,
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template,
		priv.Public(), priv)
	if err != nil {
		return nil, err
	}

	var outCert tls.Certificate
	outCert.Certificate = append(outCert.Certificate, cert)
	outCert.PrivateKey = priv

	return &outCert, nil
}
