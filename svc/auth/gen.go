package auth

import (
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/grpcmetrics"
)

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. svc.proto
//go:generate gofmt -w ./svc.pb.go

func init() {
	grpcmetrics.WrapMethods(_Auth_serviceDesc.Methods)
}

func InitMetrics(srv interface{}, mr metrics.Registry) {
	grpcmetrics.InitMetrics(srv, mr, _Auth_serviceDesc.Methods)
}
