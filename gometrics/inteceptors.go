package gometrics

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/rcrowley/go-metrics"
)

const (
	Unary        = "unary"
	ClientStream = "client_stream"
	ServerStream = "server_stream"
	BidiStream   = "bidi_stream"
)

// NewUnaryServerInterceptor returns a grpc.UnaryServerInterceptor that reports
// metrics to the go-metrics Registry provided. If prefix is not empty, it will
// be prepended to the metrics keys
func NewUnaryServerInterceptor(registry metrics.Registry, prefix string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		monitor := newMetricsReporter(registry, prefix, Unary, info.FullMethod)
		monitor.ReceivedMessage()
		resp, err := handler(ctx, req)
		monitor.Handled(grpc.Code(err))
		if err == nil {
			monitor.SentMessage()
		}
		return resp, err
	}
}

// NewStreamServerInterceptor returns a grpc.StreamServerInterceptor that
// reports metrics to the go-metrics Registry provided. If prefix is not empty,
// it will be prepended to the metrics keys
func NewStreamServerInterceptor(registry metrics.Registry, prefix string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		monitor := newMetricsReporter(registry, prefix, streamRpcType(info), info.FullMethod)
		err := handler(srv, &monitoredServerStream{ss, monitor})
		monitor.Handled(grpc.Code(err))
		return err
	}
}

func streamRpcType(info *grpc.StreamServerInfo) string {
	if info.IsClientStream && !info.IsServerStream {
		return ClientStream
	} else if !info.IsClientStream && info.IsServerStream {
		return ServerStream
	}
	return BidiStream
}

type monitoredServerStream struct {
	grpc.ServerStream
	monitor *metricsReporter
}

func (s *monitoredServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.monitor.SentMessage()
	}
	return err
}

func (s *monitoredServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.monitor.ReceivedMessage()
	}
	return err
}

type metricsReporter struct {
	rpcType     string
	serviceName string
	methodName  string
	prefix      string
	r           metrics.Registry
}

func newMetricsReporter(r metrics.Registry, prefix, rpcType, fullMethod string) *metricsReporter {
	m := &metricsReporter{r: r, rpcType: rpcType, prefix: prefix}
	split := strings.Split(fullMethod, "/")
	m.serviceName, m.methodName = split[1], split[2]
	// Number of rpc's started on the server
	metrics.GetOrRegisterCounter(fmt.Sprintf(m.prefixKey("grpc.server.started.%s.%s.%s"), rpcType, m.serviceName, m.methodName), m.r).Inc(1)
	return m
}

func (m *metricsReporter) ReceivedMessage() {
	// number of stream messages received by the server
	metrics.GetOrRegisterCounter(fmt.Sprintf(m.prefixKey("grpc.server.msgs_received.%s.%s.%s"), m.rpcType, m.serviceName, m.methodName), m.r).Inc(1)
}

func (m *metricsReporter) SentMessage() {
	// number of stream messagse sent by the server
	metrics.GetOrRegisterCounter(fmt.Sprintf(m.prefixKey("grpc.server.msgs_sent.%s.%s.%s"), m.rpcType, m.serviceName, m.methodName), m.r).Inc(1)
}

func (m *metricsReporter) Handled(code codes.Code) {
	// number of rpc calls completed on the server
	metrics.GetOrRegisterCounter(fmt.Sprintf(m.prefixKey("grpc.server.handled.%s.%s.%s.%s"), m.rpcType, m.serviceName, m.methodName, code.String()), m.r).Inc(1)
}

func (m *metricsReporter) prefixKey(key string) string {
	if m.prefix != "" {
		return fmt.Sprintf("%s.%s", m.prefix, key)
	}
	return key
}
