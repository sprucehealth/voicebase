package events

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. events.proto
//go:generate gofmt -w ./events.pb.go
