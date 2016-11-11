package invitemock

//go:generate mockgen --destination=invite.mock.go --package=invitemock github.com/sprucehealth/backend/svc/invite InviteClient
//go:generate sed -i "" -e "s|github.com/sprucehealth/backend/vendor/google.golang.org/grpc|google.golang.org/grpc|g" invite.mock.go
