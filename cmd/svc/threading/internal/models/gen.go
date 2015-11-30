package models

//go:generate protoc --gogoslick_out=. --proto_path=$GOPATH/src:. gen.proto
//go:generate gofmt -w ./gen.pb.go
