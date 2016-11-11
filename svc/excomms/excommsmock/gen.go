package excommsmock

//go:generate mockgen --destination=excomms.mock.go --package=excommsmock github.com/sprucehealth/backend/svc/excomms ExCommsClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" excomms.mock.go
