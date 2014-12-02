package apiservice

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

type SpruceError struct {
	DeveloperError     string `json:"developer_error,omitempty"`
	UserError          string `json:"user_error,omitempty"`
	DeveloperErrorCode int64  `json:"developer_code,string,omitempty"`
	HTTPStatusCode     int    `json:"-"`
	RequestID          int64  `json:"request_id,string,omitempty"`
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

func NewValidationError(msg string, r *http.Request) error {
	return &SpruceError{
		UserError:      msg,
		DeveloperError: msg,
		HTTPStatusCode: http.StatusBadRequest,
		RequestID:      GetContext(r).RequestID,
	}
}

func newJBCQForbiddenAccessError() error {
	msg := "Oops! This case has been assigned to another doctor."
	return &SpruceError{
		DeveloperErrorCode: DEVELOPER_JBCQ_FORBIDDEN,
		HTTPStatusCode:     http.StatusForbidden,
		UserError:          msg,
		DeveloperError:     msg,
	}
}

func NewAuthTimeoutError() error {
	return &SpruceError{
		UserError:          authTokenExpiredMessage,
		DeveloperErrorCode: DEVELOPER_AUTH_TOKEN_EXPIRED,
		DeveloperError:     authTokenExpiredMessage,
		HTTPStatusCode:     http.StatusForbidden,
	}
}

func NewAccessForbiddenError() error {
	msg := "Access not permitted for this information"
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
		RequestID:      GetContext(r).RequestID,
	}
}

func wrapInternalError(err error, code int, r *http.Request) error {
	return &SpruceError{
		DeveloperError: err.Error(),
		UserError:      genericUserErrorMessage,
		RequestID:      GetContext(r).RequestID,
		HTTPStatusCode: code,
	}
}

func WriteError(err error, w http.ResponseWriter, r *http.Request) {
	switch err := err.(type) {
	case *SpruceError:
		err.RequestID = GetContext(r).RequestID
		writeSpruceError(&SpruceError{
			UserError:          err.UserError,
			DeveloperError:     err.DeveloperError,
			DeveloperErrorCode: err.DeveloperErrorCode,
			HTTPStatusCode:     err.HTTPStatusCode,
			RequestID:          GetContext(r).RequestID,
		}, w, r)
	case errors.SError:
		writeSpruceError(&SpruceError{
			UserError:      err.UserError(),
			DeveloperError: err.Error(),
			HTTPStatusCode: err.HTTPStatusCode(),
			RequestID:      GetContext(r).RequestID,
		}, w, r)
	default:
		writeSpruceError(wrapInternalError(err, http.StatusInternalServerError, r).(*SpruceError), w, r)
	}
}

func WriteValidationError(msg string, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(NewValidationError(msg, r).(*SpruceError), w, r)
}

// WriteBadRequestError is used for errors that occur during parsing of the HTTP request.
func WriteBadRequestError(err error, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(wrapInternalError(err, http.StatusBadRequest, r).(*SpruceError), w, r)
}

// WriteAccessNotAllowedError is used when the user is authenticated but not
// authorized to access a given resource. Hopefully the user will never see
// this since the client shouldn't present the option to begin with.
func WriteAccessNotAllowedError(w http.ResponseWriter, r *http.Request) {
	writeSpruceError(&SpruceError{
		UserError:      "Access not permitted",
		RequestID:      GetContext(r).RequestID,
		HTTPStatusCode: http.StatusForbidden,
	}, w, r)
}

func WriteResourceNotFoundError(msg string, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(&SpruceError{
		UserError:      msg,
		RequestID:      GetContext(r).RequestID,
		HTTPStatusCode: http.StatusNotFound,
	}, w, r)
}

func writeSpruceError(err *SpruceError, w http.ResponseWriter, r *http.Request) {
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
	WriteJSONToHTTPResponseWriter(w, err.HTTPStatusCode, err)
}
