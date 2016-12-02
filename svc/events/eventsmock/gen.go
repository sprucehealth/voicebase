package eventsmock

//go:generate mockgen --destination=publisher.mock.go --package=eventsmock github.com/sprucehealth/backend/svc/events Publisher
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" publisher.mock.go

//go:generate mockgen --destination=subscriber.mock.go --package=eventsmock github.com/sprucehealth/backend/svc/events Subscriber
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" subscriber.mock.go
