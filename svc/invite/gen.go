package invite

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. svc.proto
//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. events.proto
//go:generate gofmt -w ./svc.pb.go ./events.pb.go
