package mediamock

//go:generate mockgen --destination=media.mock.go --package=mediamock github.com/sprucehealth/backend/svc/media MediaClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" media.mock.go
