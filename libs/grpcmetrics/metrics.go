package grpcmetrics

import (
	"time"

	"github.com/samuel/go-metrics/metrics"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type handlerMetrics struct {
	requests *metrics.Counter
	errors   *metrics.Counter
	latency  metrics.Histogram
}

var serviceMetrics = make(map[interface{}]map[string]*handlerMetrics) // service -> handler -> metrics

// InitMetrics creates the metrics for a specific instances of a server. It must
// be called before any service is started.
func InitMetrics(srv interface{}, mr metrics.Registry, methods []grpc.MethodDesc) {
	mets := make(map[string]*handlerMetrics)
	for _, m := range methods {
		hm := &handlerMetrics{
			requests: metrics.NewCounter(),
			errors:   metrics.NewCounter(),
			latency:  metrics.NewUnbiasedHistogram(),
		}
		mr.Add(m.MethodName+".requests", hm.requests)
		mr.Add(m.MethodName+".errors", hm.errors)
		mr.Add(m.MethodName+".latency_us", hm.latency)
		mets[m.MethodName] = hm
	}
	serviceMetrics[srv] = mets
}

// WrapMethods rewrites the handler functions to provide basic request, latency, and error metrics.
// It should only be called once for a set of methods and normally in the service definition package.
func WrapMethods(methods []grpc.MethodDesc) {
	for i, m := range methods {
		methodName := m.MethodName
		oldHandler := m.Handler
		m.Handler = func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (out interface{}, err error) {
			sm := serviceMetrics[srv]
			if sm != nil {
				hm := sm[methodName]
				hm.requests.Inc(1)
				st := time.Now()
				defer func() {
					hm.latency.Update(time.Since(st).Nanoseconds() / 1e3)
					if err != nil && (grpc.Code(err) == codes.Internal || grpc.Code(err) == codes.Unknown) {
						hm.errors.Inc(1)
					}
				}()
			}
			return oldHandler(srv, ctx, dec, interceptor)
		}
		methods[i] = m
	}
}
