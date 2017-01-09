#!/bin/sh

cd $GOPATH/src/github.com/sprucehealth/backend/cmd/svc/scheduling
go run main.go \
-debug=true \
-env=local \
-management_addr=:9021 \
-rpc_listen_addr=:50065 \