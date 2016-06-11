# GRPC Experiments

Some things I'm playing around with, easy to try out with some testing

### Polling Resolver

This is a resolver for the RoundRobin load balancer in GRPC. It
essentially takes a function that will return a list of addresses, and
an interval with which to call this. It will then provide the right
data to the balancer as hosts are added and removed.