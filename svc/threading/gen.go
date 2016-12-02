package threading

import (
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/grpcmetrics"
)

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. svc.proto
//go:generate sed -i "" s#golang.org/x/net/context#context#g ./svc.pb.go
//go:generate gofmt -w ./svc.pb.go

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. events.proto
//go:generate gofmt -w ./events.pb.go

func init() {
	grpcmetrics.WrapMethods(_Threads_serviceDesc.Methods)
}

func InitMetrics(srv interface{}, mr metrics.Registry) {
	grpcmetrics.InitMetrics(srv, mr, _Threads_serviceDesc.Methods)
}
