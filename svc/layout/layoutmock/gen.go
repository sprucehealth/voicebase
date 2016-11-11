package layoutmock

//go:generate mockgen --destination=layout.mock.go --package=layoutmock github.com/sprucehealth/backend/svc/layout LayoutClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" layout.mock.go
