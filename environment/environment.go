// Package environment provides a way in which for us to set and
// pull in the current environment stage in any package that we see it being
// necessary to do so.
package environment

const (
	Dev     = "dev"
	Prod    = "prod"
	Test    = "test"
	Staging = "staging"
	Demo    = "demo"
)

var current = Test

// SetCurrentEnvironment should be called at startup to set the current environment variable
// so as to make it possible for any package to pull in the current state to act on it
func SetCurrent(env string) {
	switch env {
	case Dev, Test, Staging, Prod, Demo:
		current = env
	default:
		panic("unexpected environment: " + env)
	}
}

func GetCurrent() string {
	return current
}

func IsDev() bool {
	return current == Dev
}

func IsProd() bool {
	return current == Prod
}

func IsDemo() bool {
	return current == Demo
}
