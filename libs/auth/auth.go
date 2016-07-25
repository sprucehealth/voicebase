package auth

import "context"

// AuthenticationProvider represents the common interface exposed but mechanisms that provide authentication
type AuthenticationProvider interface {
	Authenticate(ctx context.Context, username, password string) (id string, err error)
}
