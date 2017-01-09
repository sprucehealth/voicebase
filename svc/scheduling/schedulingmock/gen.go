package schedulingmock

//go:generate mockgen --destination=scheduling.mock.go --package=schedulingmock github.com/sprucehealth/backend/svc/scheduling SchedulingClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" scheduling.mock.go
