package auth

// AuthenticationProvider represents the common interface exposed but mechanisms that provide authentication
type AuthenticationProvider interface {
	Authenticate(username, password string) (id string, err error)
}
