package kvcertverify

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
	"time"

	"golang.org/x/net/context"

	"github.com/lstoll/grpce/reporters"

	"google.golang.org/grpc/credentials"
)

type KVStore interface {
	Get(key string) ([]byte, error)
	Put(key string, data []byte) error
	Delete(key string) error
}

type dcerttransport struct {
	store     KVStore
	clientUse bool
	// For servers
	servercreds   credentials.TransportCredentials
	serveraddress string
	validUntil    time.Time
	opts          *kvcvOpts
}

type kvcvOpts struct {
	errorReporter   reporters.ErrorReporter
	metricsReporter reporters.MetricsReporter
}

type KVCVOption func(*kvcvOpts)

func WithErrorReporter(er reporters.ErrorReporter) KVCVOption {
	return func(o *kvcvOpts) {
		o.errorReporter = er
	}
}

func WithMetricsReporter(mr reporters.MetricsReporter) KVCVOption {
	return func(o *kvcvOpts) {
		o.metricsReporter = mr
	}
}

func NewClientTransportCredentials(store KVStore, opts ...KVCVOption) credentials.TransportCredentials {
	kvcvo := &kvcvOpts{}
	for _, opt := range opts {
		opt(kvcvo)
	}
	return &dcerttransport{
		store:     store,
		clientUse: true,
		opts:      kvcvo,
	}
}

func NewServerTransportCredentials(store KVStore, address string, validUntil time.Time, opts ...KVCVOption) credentials.TransportCredentials {
	kvcvo := &kvcvOpts{}
	for _, opt := range opts {
		opt(kvcvo)
	}
	// TODO - expiry/auto-renew?

	// Generate a certificate, save it on outselves, and put it on the KV store.
	cert, err := genX509KeyPair(address, validUntil)
	if err != nil {
		reporters.ReportError(kvcvo.errorReporter, err)
		// Should we even panic here?
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
		opts:          kvcvo,
	}
}

// Clone returns a copy of the credentials
func (d *dcerttransport) Clone() credentials.TransportCredentials {
	// It's unclear what clone is for, but this should be sharing/thread safe
	return &dcerttransport{
		store:         d.store,
		clientUse:     d.clientUse,
		serveraddress: d.serveraddress,
		validUntil:    d.validUntil,
		servercreds:   d.servercreds,
		opts:          d.opts,
	}
}

// OverrideServerName overrides the server name, used before dial
func (d *dcerttransport) OverrideServerName(serverNameOverride string) error {
	d.serveraddress = serverNameOverride
	return nil
}

func (d *dcerttransport) ClientHandshake(ctx context.Context, addr string, rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	if !d.clientUse {
		return nil, nil, errors.New("Credentials not initialized for client use via NewClientDynamicCertTransportCredentials")
	}
	// Be brutal and lazy
	// Build client creds for this host
	rawCert, err := d.store.Get(addr)
	if err != nil {
		reporters.ReportError(d.opts.errorReporter, err)
		reporters.ReportCount(d.opts.metricsReporter, "kvcertverify.store.errors", 1)
		return nil, nil, err
	}
	capool := x509.NewCertPool()
	capool.AppendCertsFromPEM(rawCert)
	clientcreds := credentials.NewClientTLSFromCert(capool, addr)

	retConn, ai, err := clientcreds.ClientHandshake(ctx, addr, rawConn)
	if err != nil {
		reporters.ReportError(d.opts.errorReporter, err)
		reporters.ReportCount(d.opts.metricsReporter, "kvcertverify.client.underlyingHandshake.errors", 1)
	}
	return retConn, ai, err
}

func (d *dcerttransport) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	if d.serveraddress == "" {
		return nil, nil, errors.New("Credentials not initialized for server use via NewServerDynamicCertTransportCredentials")
	}
	retConn, ai, err := d.servercreds.ServerHandshake(rawConn)
	if err != nil {
		reporters.ReportError(d.opts.errorReporter, err)
		reporters.ReportCount(d.opts.metricsReporter, "kvcertverify.server.underlyingHandshake.errors", 1)
	}
	return retConn, ai, err
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
