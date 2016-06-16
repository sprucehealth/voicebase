package grpcmetrics_test

import (
	"errors"
	"net"
	"sort"
	"testing"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

import (
	"github.com/sprucehealth/backend/libs/grpcmetrics"
)

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. gen_test.proto
//go:generate gofmt -w ./gen_test.pb.go
//go:generate mv gen_test.pb.go gen_pb_test.go

func init() {
	grpcmetrics.WrapMethods(_Test_serviceDesc.Methods)
}

func initMetrics(srv interface{}, mr metrics.Registry) {
	grpcmetrics.InitMetrics(srv, mr, _Test_serviceDesc.Methods)
}

type testServer struct {
	metricsRegistry metrics.Registry
}

func (ts *testServer) MetricsRegistry() metrics.Registry {
	return ts.metricsRegistry
}

func (*testServer) Add(ctx context.Context, in *AddRequest) (*AddResponse, error) {
	if in.B < 0 {
		return nil, errors.New("Negatory")
	}
	return &AddResponse{Sum: in.A + in.B}, nil
}

func TestServerMetrics(t *testing.T) {
	mr := metrics.NewRegistry()

	ts := &testServer{metricsRegistry: mr}
	initMetrics(ts, mr)
	var names []string
	test.OK(t, mr.Do(func(name string, metric interface{}) error {
		names = append(names, name)
		return nil
	}))
	sort.Strings(names)
	test.Equals(t, []string{"Add.errors", "Add.latency_us", "Add.requests"}, names)

	s := grpc.NewServer()
	RegisterTestServer(s, ts)
	defer s.Stop()

	ln, err := net.Listen("tcp", "127.0.0.1:1234")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		s.Serve(ln)
	}()

	conn, err := grpc.Dial("127.0.0.1:1234", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	cli := NewTestClient(conn)

	res, err := cli.Add(context.Background(), &AddRequest{A: 100, B: 23})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", res)

	test.OK(t, mr.Do(func(name string, metric interface{}) error {
		switch name {
		case "Add.errors":
			test.Equals(t, uint64(0), metric.(*metrics.Counter).Count())
		case "Add.requests":
			test.Equals(t, uint64(1), metric.(*metrics.Counter).Count())
		}
		return nil
	}))

	_, err = cli.Add(context.Background(), &AddRequest{A: 100, B: -10})
	test.Assert(t, err != nil, "Expected non-nil error")

	test.OK(t, mr.Do(func(name string, metric interface{}) error {
		switch name {
		case "Add.errors":
			test.Equals(t, uint64(1), metric.(*metrics.Counter).Count())
		case "Add.requests":
			test.Equals(t, uint64(2), metric.(*metrics.Counter).Count())
		}
		return nil
	}))
}
