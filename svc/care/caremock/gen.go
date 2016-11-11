package caremock

//go:generate mockgen --destination=care.mock.go --package=caremock github.com/sprucehealth/backend/svc/care CareClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" care.mock.go
