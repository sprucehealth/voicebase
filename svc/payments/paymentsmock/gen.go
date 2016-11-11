package paymentsmock

//go:generate mockgen --destination=payments.mock.go --package=paymentsmock github.com/sprucehealth/backend/svc/payments PaymentsClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" payments.mock.go
