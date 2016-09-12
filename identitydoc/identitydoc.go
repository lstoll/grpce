package identitydoc

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// ErrInvalidDocument represents the failure when the document is not verified
// by the signature
var ErrInvalidDocument = errors.New("The provided identify document does not match the signature")

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
	InstanceType     string    `json:"instanceType"`
	ImageID          string    `json:"imageId"`
}

// VerifyDocumentAndSignature will confirm that the document is correct by
// validating it against the signature and cert for the given region. It will
// return the parsed document if it's valid, or ErrInvalidDocument if it's not.
// The document can be retrieved from:
// http://169.254.169.254/latest/dynamic/instance-identity/document
// And the signature:
// http://169.254.169.254/latest/dynamic/instance-identity/signature
func VerifyDocumentAndSignature(region string, document, signature []byte) (*InstanceIdentityDocument, error) {
	c, err := CertForRegion(region)
	if err != nil {
		return nil, err
	}

	ds, err := base64.StdEncoding.DecodeString(string(signature))
	if err != nil {
		return nil, err
	}

	err = c.CheckSignature(x509.SHA256WithRSA, document, []byte(ds))
	if err != nil {
		return nil, ErrInvalidDocument
	}

	iid := &InstanceIdentityDocument{}
	err = json.Unmarshal(document, iid)
	if err != nil {
		return nil, err
	}
	return iid, nil
}
