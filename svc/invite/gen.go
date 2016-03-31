package invite

import (
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/grpcmetrics"
)

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. svc.proto
//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. events.proto
//go:generate gofmt -w ./svc.pb.go ./events.pb.go

func init() {
	grpcmetrics.WrapMethods(_Invite_serviceDesc.Methods)
}

func InitMetrics(srv interface{}, mr metrics.Registry) {
	grpcmetrics.InitMetrics(srv, mr, _Invite_serviceDesc.Methods)
}
