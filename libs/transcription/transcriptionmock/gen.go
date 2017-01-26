package transcriptionmock

//go:generate mockgen --destination=transcription.mock.go --package=transcriptionmock github.com/sprucehealth/backend/libs/transcription Provider
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" transcription.mock.go
