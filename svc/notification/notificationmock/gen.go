package notificationmock

//go:generate mockgen --destination=notification.mock.go --package=notificationmock github.com/sprucehealth/backend/svc/notification Client
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" notification.mock.go
