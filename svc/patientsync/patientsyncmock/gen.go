package patientsyncmock

//go:generate mockgen --destination=patientsync.mock.go --package=patientsyncmock github.com/sprucehealth/backend/svc/patientsync PatientSyncClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" patientsync.mock.go
