package media

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. svc.proto
//go:generate gofmt -w ./svc.pb.go
