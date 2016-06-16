package apiservice

type BadRequestError interface {
	IsBadRequestErr()
	Error() string
}

type ErrBadRequest struct{ err error }

func (e *ErrBadRequest) IsBadRequestErr() {}
func (e *ErrBadRequest) Error() string    { return e.err.Error() }

func NewBadRequestError(err error) BadRequestError {
	return &ErrBadRequest{err}
}

func IsBadRequestError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(BadRequestError)
	return ok
}
