package events

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. events.proto
//go:generate sed -i "" s#golang.org/x/net/context#context#g ./events.pb.go
//go:generate gofmt -w ./events.pb.go
