package settingsmock

//go:generate mockgen --destination=settings.mock.go --package=settingsmock github.com/sprucehealth/backend/svc/settings SettingsClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" settings.mock.go
