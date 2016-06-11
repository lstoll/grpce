# GRPC Experiments

Some things I'm playing around with, easy to try out with some testing

### Polling Resolver

This is a resolver for the RoundRobin load balancer in GRPC. It
essentially takes a function that will return a list of addresses, and
an interval with which to call this. It will then provide the right
data to the balancer as hosts are added and removed.

### Dynamic certs

This is a TLS credential implementation for the server that generates
a self-signed cert on the fly. This will get persisted in a KV
store. The client implementation looks up the cert based on the
address from the KV store, and validates it at the root cert for this
connection.

### Instance Identity Document Transport Auth.

This ia a credentials.TransportCredentials implementation intended to
wrap another transport (e.g dynamic certs). On top of this it adds a
method for verifying the connecting client via it's
[Instance Identity Document](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html). When
connecting to a server, the client will fetch the document and pkcs7
signature fromt the Amazon Instance Identity server, then encode this
and pass it to the server. On the server side, the transport will read
this info from the client post connection. It will then check the
document & signature against AWS's Cert. If the data is missing,
forged or not signed by AWS the connection will be dropped.
