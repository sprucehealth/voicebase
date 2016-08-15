package sync

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. sync.proto
//go:generate sed -i "" s#golang.org/x/net/context#context#g ./sync.pb.go
//go:generate gofmt -w ./sync.pb.go
