package authmock

//go:generate mockgen --destination=auth.mock.go --package=authmock github.com/sprucehealth/backend/svc/auth AuthClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" auth.mock.go
