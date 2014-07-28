package errors

// UserError interface makes it possible to easily indicate what errors are meant to be
// shown to the user
type UserError interface {
	IsUserError() bool
	UserError() string
	Error() string
}
