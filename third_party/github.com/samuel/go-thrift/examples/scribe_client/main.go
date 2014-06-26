package main

import (
	"fmt"
	"net"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-thrift/examples/scribe"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-thrift/thrift"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:1463")
	if err != nil {
		panic(err)
	}

	client := thrift.NewClient(thrift.NewFramedReadWriteCloser(conn, 0), thrift.NewBinaryProtocol(true, false), false)
	scr := scribe.ScribeClient{Client: client}
	res, err := scr.Log([]*scribe.LogEntry{{"category", "message"}})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Response: %+v\n", res)
}
