package identitydoc

import "testing"

var instance1Pkcs7 = `MIAGCSqGSIb3DQEHAqCAMIACAQExCzAJBgUrDgMCGgUAMIAGCSqGSIb3DQEHAaCAJIAEggGwewog
ICJwcml2YXRlSXAiIDogIjE3Mi4zMS41NC4yNDkiLAogICJkZXZwYXlQcm9kdWN0Q29kZXMiIDog
bnVsbCwKICAiYXZhaWxhYmlsaXR5Wm9uZSIgOiAidXMtZWFzdC0xYSIsCiAgInZlcnNpb24iIDog
IjIwMTAtMDgtMzEiLAogICJpbnN0YW5jZUlkIiA6ICJpLTBlOTBkNDk0ZWNmMWVhNGJjIiwKICAi
YmlsbGluZ1Byb2R1Y3RzIiA6IG51bGwsCiAgImluc3RhbmNlVHlwZSIgOiAidDIubWljcm8iLAog
ICJhY2NvdW50SWQiIDogIjcxMjM0OTg2MDg4MyIsCiAgImltYWdlSWQiIDogImFtaS1mY2UzYzY5
NiIsCiAgInBlbmRpbmdUaW1lIiA6ICIyMDE2LTA2LTA5VDAyOjI1OjI4WiIsCiAgImFyY2hpdGVj
dHVyZSIgOiAieDg2XzY0IiwKICAia2VybmVsSWQiIDogbnVsbCwKICAicmFtZGlza0lkIiA6IG51
bGwsCiAgInJlZ2lvbiIgOiAidXMtZWFzdC0xIgp9AAAAAAAAMYIBGDCCARQCAQEwaTBcMQswCQYD
VQQGEwJVUzEZMBcGA1UECBMQV2FzaGluZ3RvbiBTdGF0ZTEQMA4GA1UEBxMHU2VhdHRsZTEgMB4G
A1UEChMXQW1hem9uIFdlYiBTZXJ2aWNlcyBMTEMCCQCWukjZ5V4aZzAJBgUrDgMCGgUAoF0wGAYJ
KoZIhvcNAQkDMQsGCSqGSIb3DQEHATAcBgkqhkiG9w0BCQUxDxcNMTYwNjA5MDIyNTMyWjAjBgkq
hkiG9w0BCQQxFgQUYQi1EvdojKr1UkbWb8e3PRfTefEwCQYHKoZIzjgEAwQvMC0CFGUvzOyBtGGl
a40lrRD8473cC947AhUAsJDK8QzfImuoWdIpfKtatcYKX2kAAAAAAAA=`

var instance1Document = `{
  "privateIp" : "172.31.54.249",
  "devpayProductCodes" : null,
  "availabilityZone" : "us-east-1a",
  "version" : "2010-08-31",
  "instanceId" : "i-0e90d494ecf1ea4bc",
  "billingProducts" : null,
  "instanceType" : "t2.micro",
  "accountId" : "712349860883",
  "imageId" : "ami-fce3c696",
  "pendingTime" : "2016-06-09T02:25:28Z",
  "architecture" : "x86_64",
  "kernelId" : null,
  "ramdiskId" : null,
  "region" : "us-east-1"
}`

var instance2Pkcs7 = `MIAGCSqGSIb3DQEHAqCAMIACAQExCzAJBgUrDgMCGgUAMIAGCSqGSIb3DQEHAaCAJIAEggGwewog
ICJwcml2YXRlSXAiIDogIjE3Mi4zMS41Ni4yNDUiLAogICJkZXZwYXlQcm9kdWN0Q29kZXMiIDog
bnVsbCwKICAiYXZhaWxhYmlsaXR5Wm9uZSIgOiAidXMtZWFzdC0xYSIsCiAgInZlcnNpb24iIDog
IjIwMTAtMDgtMzEiLAogICJpbnN0YW5jZUlkIiA6ICJpLTA2NmNmNzA5Mzk3Yjc4N2Q5IiwKICAi
YmlsbGluZ1Byb2R1Y3RzIiA6IG51bGwsCiAgImluc3RhbmNlVHlwZSIgOiAidDIubWljcm8iLAog
ICJhY2NvdW50SWQiIDogIjcxMjM0OTg2MDg4MyIsCiAgInBlbmRpbmdUaW1lIiA6ICIyMDE2LTA2
LTA5VDAyOjM3OjQ1WiIsCiAgImltYWdlSWQiIDogImFtaS1mY2UzYzY5NiIsCiAgImFyY2hpdGVj
dHVyZSIgOiAieDg2XzY0IiwKICAia2VybmVsSWQiIDogbnVsbCwKICAicmFtZGlza0lkIiA6IG51
bGwsCiAgInJlZ2lvbiIgOiAidXMtZWFzdC0xIgp9AAAAAAAAMYIBGDCCARQCAQEwaTBcMQswCQYD
VQQGEwJVUzEZMBcGA1UECBMQV2FzaGluZ3RvbiBTdGF0ZTEQMA4GA1UEBxMHU2VhdHRsZTEgMB4G
A1UEChMXQW1hem9uIFdlYiBTZXJ2aWNlcyBMTEMCCQCWukjZ5V4aZzAJBgUrDgMCGgUAoF0wGAYJ
KoZIhvcNAQkDMQsGCSqGSIb3DQEHATAcBgkqhkiG9w0BCQUxDxcNMTYwNjA5MDIzNzUwWjAjBgkq
hkiG9w0BCQQxFgQUDh1ZkzRJpKAc7tXjV1ClM5vqi6AwCQYHKoZIzjgEAwQvMC0CFF8mMVcMvUnW
sfF1Z1N9W/O2YgtqAhUAkJpYBglzpwx8u8waHABgBhi+PmoAAAAAAAA=`

var instance2Document = `{
  "privateIp" : "172.31.56.245",
  "devpayProductCodes" : null,
  "availabilityZone" : "us-east-1a",
  "version" : "2010-08-31",
  "instanceId" : "i-066cf709397b787d9",
  "billingProducts" : null,
  "instanceType" : "t2.micro",
  "accountId" : "712349860883",
  "pendingTime" : "2016-06-09T02:37:45Z",
  "imageId" : "ami-fce3c696",
  "architecture" : "x86_64",
  "kernelId" : null,
  "ramdiskId" : null,
  "region" : "us-east-1"
}`

// We can use this for validating against an "invalid cert"
var comdirectCert = `-----BEGIN CERTIFICATE-----
MIIFpjCCBI6gAwIBAgIJAIrHIweEGQCQMA0GCSqGSIb3DQEBBQUAMGoxCzAJBgNV
BAYTAkRFMTIwMAYDVQQKDClTQ0EgRGV1dHNjaGUgUG9zdCBTaWdudHJ1c3QgdW5k
IERNREEgR21iSDEnMCUGA1UEAwweU2lnbnRydXN0IENFUlQgQ2xhc3MgMiBDQSA3
OlBOMB4XDTE1MDEyMjEyMDU0OFoXDTE5MDEyMjEyMDU0N1owgekxITAfBgkqhkiG
9w0BCQEWEmt1bmRlQGNvbWRpcmVjdC5kZTEiMCAGA1UEAwwZY29tZGlyZWN0IEt1
bmRlbmJldHJldXVuZzELMAkGA1UEBhMCREUxEjAQBgNVBAcMCVF1aWNrYm9ybjEb
MBkGA1UECAwSU2NobGVzd2lnLUhvbHN0ZWluMRowGAYDVQQKDBFDb21kaXJlY3Qg
QmFuayBBRzElMCMGA1UECwwcKEUtTWFpbCBHYXRld2F5IENlcnRpZmljYXRlKTEf
MB0GA1UEBRMWMDAxMDAwMDAwMDAxMjc4NzQ1MDAwMDCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBAMe6DX2MgA0+gp5G1jyUxqyUlczFvI9kM9310LhX8Kbm
pDDKsi/49Qns3N3lB7Ke6eBZMrSBlXrukf4JJQrJSRRqdneEgam9BRt62hL5njes
Nz0DKlmw0I3w+Lznh5NfU0N9MXh+0uG1FkMijs+Lj2XlZfapa2rcCGqrBgxJpM24
+0ZHwL2RAhDJy6D/MLmyBTNY72mzrJ40r0asOT+RPyIoB8Jv/dHAgt37VIHuFf1q
pilv799Tr6E4XPz6wHq7JUd0VMj7jjhbFKbpfhRIrnNOjIpwxI+77bjdpdc0mlUm
0pWAfHHzHQ2/3N0u/Prdd9I3250qYBaK+XgzXNdrJsECAwEAAaOCAc0wggHJMB0G
A1UdDgQWBBRNSggREK1dQKXAxbdIVcXX4+yK+TAOBgNVHQ8BAf8EBAMCBLAwFAYD
VR0gBA0wCzAJBgcrEgkCAgIBMAwGA1UdEwEB/wQCMAAwHQYDVR0lBBYwFAYIKwYB
BQUHAwIGCCsGAQUFBwMEMIHHBgNVHR8Egb8wgbwwgbmgR6BFhkNodHRwOi8vd3d3
LnNpZ250cnVzdC5kZS9jcmwvZHBzaWdudHJ1c3QvbnFzaWcvc3RjZXJ0X2NsYXNz
Ml9jYTcuY3Jsom6kbDBqMQswCQYDVQQGEwJERTEyMDAGA1UECgwpU0NBIERldXRz
Y2hlIFBvc3QgU2lnbnRydXN0IHVuZCBETURBIEdtYkgxJzAlBgNVBAMMHlNpZ250
cnVzdCBDRVJUIENsYXNzIDIgQ0EgNzpQTjBLBggrBgEFBQcBAQQ/MD0wOwYIKwYB
BQUHMAGGL2h0dHA6Ly9vY3NwLnNpZ250cnVzdC5kZS9vY3NwL2Rwc2lnbnRydXN0
L25xc2lnMB0GA1UdEQQWMBSBEmt1bmRlQGNvbWRpcmVjdC5kZTAfBgNVHSMEGDAW
gBS7B4qkUNWEq6OMi8ODXTNQrc/PqjANBgkqhkiG9w0BAQUFAAOCAQEABJbf1fLI
eJxu8alfqlaKqVDrYXSnAqapbbJInLSD6DlpddaBFDaL7Uul2TyavLWHcViYOpqQ
z3FonxkVKbAOg7dSbWFIsgD7wJ6j5h+Hcev9n299+z0c5cEvtURBL4zaNOEIgJuh
TIQAnnS7yBpH9qWMpiCv1wUF1SbQniFLiKGo+pxuddp92JWh0wXCFguf8kjDdhIz
SoeFoi4ipg+tIKrm7tI79EWlgcQFb+9UBuh1QbadAxJK3ON1D0EjxFvUnDf/CeIa
+RBz5aLtis1efTPHjk02fAlHaAIgEA+vibz/FMOL3iIjTnMXv1yxwThRrwwKRToa
0p6VwEd1Ddx7NQ==
-----END CERTIFICATE-----`

func TestDocVerification(t *testing.T) {
	if !verifyDocToPKCS7([]byte(instance1Document), []byte(instance1Pkcs7)) {
		t.Error("Instance 1 valid docs failed auth")
	}

	if verifyDocToPKCS7([]byte(instance1Document), []byte(instance2Pkcs7)) {
		t.Error("Instance 1 doc with instance 2 pkcs7 passed auth")
	}

	origCert := awsPubKey
	awsPubKey = comdirectCert
	if verifyDocToPKCS7([]byte(instance1Document), []byte(instance1Pkcs7)) {
		t.Error("Instance 1 doc and sig passed with comdirect cert")
	}
	awsPubKey = origCert
}

/*func TestHTTPClient(t *testing.T) {
	instanceDoc, pkcs7 := instance1Document, instance1Pkcs7
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/doc", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, instanceDoc)
	})
	serveMux.HandleFunc("/pkcs7", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, pkcs7)
	})
	go http.ListenAndServe("127.0.0.1:15800", serveMux)
	identityDocURL = "http://127.0.0.1:15800/doc"
	pkcs7URL = "http://127.0.0.1:15800/pkcs7"

}*/
