package operational

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. svc.proto
//go:generate sed -i "" s#golang.org/x/net/context#context#g ./svc.pb.go
//go:generate gofmt -w ./svc.pb.go
