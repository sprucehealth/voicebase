package threadingmock

//go:generate mockgen --destination=threading.mock.go --package=threadingmock github.com/sprucehealth/backend/svc/threading ThreadsClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" threading.mock.go
