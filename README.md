# GRPC Experiments for AWS/EC2

This repo has some experimental components for running services in a dynamic environment inside EC2. It's targeted at grpc-go, but would be easy to adapt to pretty much anything. It's intended to work reliably with litte infrastructure overhead, and without just having one magic shared cert/key everyone uses. The only external dependency is a KV store, which I intend to use S3. S3 is simple, reliable, and likely effective enough for this. Load balancing is handled by the client which eliminates the need for an ELB. Servers generate their own certificates and store the public component in the KV store, to avoid needing a CA infrastructure. This also still allows the revolation of per-server creds, and verifying that the server you connect to is the exact server. Client auth is handled by [Instance Identity Documents](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html). This allows us to assert the account ID and instance ID of clients, without requiring any pre-shared state. This means instances can be managed inside an ASG, but not using a long-lasting shared key or other external service. All service discover is done via the KV store (i.e S3), with a model that is forgiving of it's consistency model

The end result should be a way to securely run services inside AWS without any long lasting or shared credentials, without requiring any infrastructure over other than a S3 bucket and some IAM/role policies.

## Overall flow

![Diagram representing flow in system](https://cdn.lstoll.net/screen/grpcexperiments_flow.html_-_draw.io_2016-06-11_14-29-17.png)

## Components

### Polling Resolver

This is a resolver for the RoundRobin load balancer in GRPC. It
essentially takes a function that will return a list of addresses, and
an interval with which to call this. It will then provide the right
data to the balancer as hosts are added and removed.

```go
// This is the function we'll poll for the current list of servers
lookup := func(key string) ([]string, error) {
	if key == "testtarget" {
		return []string{"server1.abc.com", "server2.abc.com"}, nil
	}
	return nil, fmt.Errorf("Unknown target: %q", key)
}

conn, err := grpc.Dial("testtarget",
	grpc.WithBalancer(grpc.RoundRobin(NewPollingResolver("testtarget", 10*time.Second, lookup))))
```

### Dynamic certs

This is a TLS credential implementation for the server that generates
a self-signed cert on the fly. This will get persisted in a KV
store. The client implementation looks up the cert based on the
address from the KV store, and validates it at the root cert for this
connection.

```go
// store matches a Get/Put/Delete interface.
s := grpc.NewServer(grpc.Creds(NewServerDynamicCertTransportCredentials(store, address, time.Now().AddDate(0, 0, 1))))

// The client gets the same store.
conn, err := grpc.Dial(address, grpc.WithTransportCredentials(NewClientDynamicCertTransportCredentials(store)))
```

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
forged or not signed by AWS the connection will be dropped. The
function `InstanceIdentityDocumentAuthInfoFromContext` is also
provided. This can be used in the server implementation to get the
document information, which can then be used to authorize against the
instance or account ID.

```go
// This wraps an existing transport, and layers the doc auth on top.
s := grpc.NewServer(grpc.Creds(
	NewInstanceAuthTransportCredentials(
		NewServerDynamicCertTransportCredentials(store, address, time.Now().AddDate(0, 0, 1)),
	),
))

// Same wrapping style on the client
conn, err := grpc.Dial(address, grpc.WithTransportCredentials(
	NewInstanceAuthTransportCredentials(
		NewClientDynamicCertTransportCredentials(store)),
))

// In your server method, you can retrieve info about the connecting client
func (t *authtpserver) GetLBInfo(ctx context.Context, req *testproto.LBInfoRequest) (*testproto.LBInfoResponse, error) {
	ai, err := InstanceIdentityDocumentAuthInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return &testproto.LBInfoResponse{
		Name: ai.InstanceID,
	}, nil
}

```

## Security

The instance identity document security model and code hasn't really been audited or had deep thought put in to it. The concept works in my head, but I need an external opinion and to think about all the risks. The certificate storage is potentially easier to reason about, the model for server verification depends on managing access to write and update to the KV store.
