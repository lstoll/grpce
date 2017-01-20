package identitydoc

import (
	"bytes"
	"testing"
)

var testSig = `Ob3mEexQi/91fA/HMqS7L1DraJ/8T/lAblai/PrSgx6FMMPpQpi2rftc/iUcs4Uufzq0NjXkwk95
9cRES6s3T36hWgob/cutg5imhdy5++bymuzE8Z6T35pU3y3kn4eS6Yebna1atVbAFifeAqySGXCZ
l5+VTbjj/MBI7vB1cEs=`

var testDoc = `{
  "devpayProductCodes" : null,
  "privateIp" : "172.30.0.208",
  "availabilityZone" : "us-east-1a",
  "accountId" : "021124591875",
  "version" : "2010-08-31",
  "instanceId" : "i-1ddaabe5",
  "billingProducts" : null,
  "instanceType" : "t2.nano",
  "pendingTime" : "2016-09-03T15:07:45Z",
  "architecture" : "x86_64",
  "imageId" : "ami-2d39803a",
  "kernelId" : null,
  "ramdiskId" : null,
  "region" : "us-east-1"
}`

func TestDocVerification(t *testing.T) {
	doc, err := VerifyDocumentAndSignature("us-east-1", []byte(testDoc), []byte(testSig))
	if err != nil {
		t.Errorf("Error validating document %q", err)
	}
	if doc == nil {
		t.Error("Test document failed auth")
	}

	if !bytes.Equal(doc.Doc, []byte(testDoc)) {
		t.Error("Raw document did not match input document")
	}
	if !bytes.Equal(doc.Sig, []byte(testSig)) {
		t.Error("Document signature did not match input signature")
	}

	mod := testDoc + "lol"
	doc, err = VerifyDocumentAndSignature("us-east-1", []byte(mod), []byte(testSig))
	if err != ErrInvalidDocument {
		t.Error("Invalid document didn't return ErrInvalidDocument")
	}
	if doc != nil {
		t.Error("Invalid, errored document did not return nil")
	}

	regions := []string{"us-east-1", "us-west-1", "us-west-2", "ap-southeast-2",
		"ap-southeast-1", "ap-northeast-1", "eu-central-1", "eu-west-1",
		"sa-east-1"}
	// "ap-south-1" and "ap-northeast-2" are currently unsupported
	for _, r := range regions {
		_, err := VerifyDocumentAndSignature(r, []byte{}, []byte{})
		if err == ErrUnknownRegion {
			t.Errorf("Region %s returned ErrUnknownRegion", r)
		}
	}
}
