package identitydoc

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// ErrInvalidDocument represents the failure when the document is not verified
// by the signature
var ErrInvalidDocument = errors.New("The provided identify document does not match the signature")

var (
	// IdentityDocURL is the URL to retrieve the identity doc from
	IdentityDocURL = "http://169.254.169.254/latest/dynamic/instance-identity/document"
	// SignatureURL is the URL to retrieve the signed identity doc from
	SignatureURL = "http://169.254.169.254/latest/dynamic/instance-identity/signature"
)

// InstanceIdentityDocument represents the information containe in an instances
// identity document
// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html
type InstanceIdentityDocument struct {
	InstanceID       string    `json:"instanceId"`
	AccountID        string    `json:"accountId"`
	PrivateIP        string    `json:"privateIp"`
	Region           string    `json:"region"`
	AvailabilityZone string    `json:"availabilityZone"`
	PendingTime      time.Time `json:"pendingTime"`
}

// GetDocumentAndSignature will return the document and it's signature from the
// metadata API
func GetDocumentAndSignature() (doc, p7sig []byte, err error) {
	doc, err = httpGET(IdentityDocURL, 10)
	if err != nil {
		return
	}
	p7sig, err = httpGET(SignatureURL, 10)
	if err != nil {
		return
	}
	return
}

// VerifyDocumentAndSignature will confirm that the document is correct by
// validating it against the signature and cert for the given region. It will
// return the parsed document if it's valid, or ErrInvalidDocument if it's not.
func VerifyDocumentAndSignature(region string, doc, sig []byte) (*InstanceIdentityDocument, error) {
	c, err := CertForRegion(region)
	if err != nil {
		return nil, err
	}

	ds, err := base64.StdEncoding.DecodeString(string(sig))
	if err != nil {
		return nil, err
	}

	err = c.CheckSignature(x509.SHA256WithRSA, doc, []byte(ds))
	if err != nil {
		return nil, ErrInvalidDocument
	}

	iid := &InstanceIdentityDocument{}
	err = json.Unmarshal(doc, iid)
	if err != nil {
		return nil, err
	}
	return iid, nil
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
