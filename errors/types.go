package errors

// SError interface makes it possible for any package to describe an error
// without having to depend on the utility methods in this package
type SError interface {
	IsUserError() bool
	UserError() string
	Error() string
	HTTPStatusCode() int
}
