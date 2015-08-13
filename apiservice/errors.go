package apiservice

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

// SError interface makes it possible for any package to describe an error
// without having to depend on the utility methods in this package
type SError interface {
	IsUserError() bool
	UserError() string
	Error() string
	HTTPStatusCode() int
}

type SpruceError struct {
	DeveloperError     string `json:"developer_error,omitempty"`
	UserError          string `json:"user_error,omitempty"`
	DeveloperErrorCode int64  `json:"developer_code,string,omitempty"`
	HTTPStatusCode     int    `json:"-"`
	RequestID          uint64 `json:"request_id,string,omitempty"`
}

func (s *SpruceError) Error() string {
	var msg string
	e := s.DeveloperError
	if e == "" {
		e = s.UserError
	}
	if s.DeveloperErrorCode > 0 {
		msg = fmt.Sprintf("RequestID: %d, Error: %s, ErrorCode: %d, StatusCode: %d", s.RequestID, e, s.DeveloperErrorCode, s.HTTPStatusCode)
	} else {
		msg = fmt.Sprintf("RequestID: %d, Error: %s, StatusCode: %d", s.RequestID, e, s.HTTPStatusCode)
	}
	return msg
}

func NewError(msg string, httpStatusCode int) error {
	return &SpruceError{
		UserError:      msg,
		DeveloperError: msg,
		HTTPStatusCode: httpStatusCode,
	}
}

func NewValidationError(msg string) error {
	return &SpruceError{
		UserError:      msg,
		DeveloperError: msg,
		HTTPStatusCode: http.StatusBadRequest,
	}
}

func newJBCQForbiddenAccessError() error {
	msg := "Oops! This case has been assigned to another doctor."
	return &SpruceError{
		DeveloperErrorCode: DeveloperErrorJBCQForbidden,
		HTTPStatusCode:     http.StatusForbidden,
		UserError:          msg,
		DeveloperError:     msg,
	}
}

func NewAuthTimeoutError() error {
	return &SpruceError{
		UserError:          authTokenExpiredMessage,
		DeveloperErrorCode: DeveloperErrorAuthTokenExpired,
		DeveloperError:     authTokenExpiredMessage,
		HTTPStatusCode:     http.StatusForbidden,
	}
}

func NewAccessForbiddenError() error {
	msg := "Access not permitted"
	return &SpruceError{
		HTTPStatusCode: http.StatusForbidden,
		UserError:      msg,
		DeveloperError: msg,
	}
}

func NewCareCoordinatorAccessForbiddenError() error {
	msg := "Care Coordinator can only view patient file and case information or interact with patient via messaging."
	return &SpruceError{
		UserError:      msg,
		DeveloperError: msg,
		HTTPStatusCode: http.StatusForbidden,
	}
}

func NewResourceNotFoundError(msg string, r *http.Request) error {
	return &SpruceError{
		UserError:      msg,
		HTTPStatusCode: http.StatusNotFound,
	}
}

func wrapInternalError(err error, code int, r *http.Request) error {
	return &SpruceError{
		DeveloperError: err.Error(),
		UserError:      genericUserErrorMessage,
		HTTPStatusCode: code,
	}
}

func WriteError(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) {
	switch err := err.(type) {
	case *SpruceError:
		writeSpruceError(ctx, err, w, r)
	case SError:
		writeSpruceError(ctx, &SpruceError{
			UserError:      err.UserError(),
			DeveloperError: err.Error(),
			HTTPStatusCode: err.HTTPStatusCode(),
		}, w, r)
	default:
		writeSpruceError(ctx, wrapInternalError(err, http.StatusInternalServerError, r).(*SpruceError), w, r)
	}
}

func WriteErrorResponse(w http.ResponseWriter, httpStatusCode int, errorResponse ErrorResponse) {
	golog.LogDepthf(1, golog.ERR, errorResponse.DeveloperError)
	httputil.JSONResponse(w, httpStatusCode, &errorResponse)
}

func WriteDeveloperErrorWithCode(w http.ResponseWriter, developerStatusCode int64, httpStatusCode int, errorString string) {
	golog.LogDepthf(1, golog.WARN, errorString)
	developerError := &ErrorResponse{
		DeveloperError: errorString,
		DeveloperCode:  developerStatusCode,
		UserError:      genericUserErrorMessage,
	}
	httputil.JSONResponse(w, httpStatusCode, developerError)
}

func WriteUserError(w http.ResponseWriter, httpStatusCode int, errorString string) {
	userError := &ErrorResponse{
		UserError: errorString,
	}
	httputil.JSONResponse(w, httpStatusCode, userError)
}

func WriteValidationError(ctx context.Context, msg string, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(ctx, NewValidationError(msg).(*SpruceError), w, r)
}

// WriteBadRequestError is used for errors that occur during parsing of the HTTP request.
func WriteBadRequestError(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(ctx, wrapInternalError(err, http.StatusBadRequest, r).(*SpruceError), w, r)
}

// WriteAccessNotAllowedError is used when the user is authenticated but not
// authorized to access a given resource. Hopefully the user will never see
// this since the client shouldn't present the option to begin with.
func WriteAccessNotAllowedError(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(ctx, &SpruceError{
		UserError:      "Access not permitted",
		HTTPStatusCode: http.StatusForbidden,
	}, w, r)
}

func WriteResourceNotFoundError(ctx context.Context, msg string, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(ctx, &SpruceError{
		UserError:      msg,
		HTTPStatusCode: http.StatusNotFound,
	}, w, r)
}

func writeSpruceError(ctx context.Context, err *SpruceError, w http.ResponseWriter, r *http.Request) {
	err.RequestID = httputil.RequestID(ctx)
	var msg = err.DeveloperError
	if msg == "" {
		msg = err.UserError
	}
	lvl := golog.INFO
	if err.HTTPStatusCode == http.StatusInternalServerError {
		lvl = golog.ERR
	}
	golog.Context(
		"RequestID", err.RequestID,
		"ErrorCode", err.DeveloperErrorCode,
	).LogDepthf(2, lvl, msg)

	// remove the developer error information if we are not dealing with
	// before sending information across the wire
	if !environment.IsDev() {
		err.DeveloperError = ""
	}
	httputil.JSONResponse(w, err.HTTPStatusCode, err)
}
