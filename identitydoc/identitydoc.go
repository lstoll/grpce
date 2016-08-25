package identitydoc

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/lstoll/pkcs7"
)

var ErrInvalidDocument = fmt.Errorf("The provided identify document does not match the signature")

var (
	// IdentityDocURL is the URL to retrieve the identity doc from
	IdentityDocURL = "http://169.254.169.254/latest/dynamic/instance-identity/document"
	// Pkcs7URL is the URL to retrieve the signed identity doc from
	Pkcs7URL = "http://169.254.169.254/latest/dynamic/instance-identity/pkcs7"
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

type InstanceIdentityDocument struct {
	InstanceID       string    `json:"instanceId"`
	AccountID        string    `json:"accountId"`
	PrivateIP        string    `json:"privateIp"`
	Region           string    `json:"region"`
	AvailabilityZone string    `json:"availabilityZone"`
	PendingTime      time.Time `json:"pendingTime"`
}

func GetDocumentAndSignature() (doc, p7sig []byte, err error) {
	// Load the identity doc & sig
	doc, err = httpGET(IdentityDocURL, 10)
	if err != nil {
		return
	}
	p7sig, err = httpGET(Pkcs7URL, 10)
	if err != nil {
		return
	}
	return
}

func VerifyDocumentAndSignature(doc, p7sig []byte) (*InstanceIdentityDocument, error) {
	if !verifyDocToPKCS7(doc, p7sig) {
		return nil, ErrInvalidDocument
	}

	iid := &InstanceIdentityDocument{}
	err := json.Unmarshal(doc, iid)
	if err != nil {
		return nil, err
	}
	return iid, nil
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
