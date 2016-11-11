package directorymock

//go:generate mockgen --destination=directory.mock.go --package=directorymock github.com/sprucehealth/backend/svc/directory DirectoryClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" directory.mock.go
