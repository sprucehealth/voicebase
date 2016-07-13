package deploy

import (
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/grpcmetrics"
)

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. svc.proto
//go:generate sed -i "" s#golang.org/x/net/context#context#g ./svc.pb.go
//go:generate gofmt -w ./svc.pb.go

func init() {
	grpcmetrics.WrapMethods(_Deploy_serviceDesc.Methods)
}

func InitMetrics(srv interface{}, mr metrics.Registry) {
	grpcmetrics.InitMetrics(srv, mr, _Deploy_serviceDesc.Methods)
}
