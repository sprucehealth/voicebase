package models

//go:generate protoc --gogoslick_out=plugins=grpc:. --proto_path=$GOPATH/src:. models.proto
//go:generate gofmt -w ./models.pb.go
