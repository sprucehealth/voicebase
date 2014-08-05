package apiservice

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

type spruceError struct {
	DeveloperError     string `json:"developer_error,omitempty"`
	UserError          string `json:"user_error,omitempty"`
	DeveloperErrorCode int64  `json:"developer_code,string,omitempty"`
	HTTPStatusCode     int    `json:"-"`
	RequestID          int64  `json:"request_id,string,omitempty"`
}

func (s *spruceError) Error() string {
	var msg string
	if s.DeveloperErrorCode > 0 {
		msg = fmt.Sprintf("RequestID: %d, Error: %s, ErrorCode: %d", s.RequestID, s.DeveloperError, s.DeveloperErrorCode)
	} else {
		msg = fmt.Sprintf("RequestID: %d, Error: %s", s.RequestID, s.DeveloperError)
	}
	return msg
}

func NewValidationError(msg string, r *http.Request) error {
	return &spruceError{
		UserError:      msg,
		DeveloperError: msg,
		HTTPStatusCode: http.StatusBadRequest,
		RequestID:      GetContext(r).RequestID,
	}
}

func newJBCQForbiddenAccessError() error {
	msg := "Oops! This case has been assigned to another doctor."
	return &spruceError{
		DeveloperErrorCode: DEVELOPER_JBCQ_FORBIDDEN,
		HTTPStatusCode:     http.StatusForbidden,
		UserError:          msg,
		DeveloperError:     msg,
	}
}

func NewAccessForbiddenError() *spruceError {
	msg := "Access not permitted for this information"
	return &spruceError{
		HTTPStatusCode: http.StatusForbidden,
		UserError:      msg,
		DeveloperError: msg,
	}
}

func NewCareCoordinatorAccessForbiddenError() error {
	return &spruceError{
		UserError:      "Care Coordinator can only view patient file and case information or interact with patient via messaging.",
		DeveloperError: "Care Coordinator can only view patient file and case information or interact with patient via messaging.",
		HTTPStatusCode: http.StatusForbidden,
	}
}

func NewResourceNotFoundError(msg string, r *http.Request) error {
	return &spruceError{
		UserError:      msg,
		HTTPStatusCode: http.StatusNotFound,
		RequestID:      GetContext(r).RequestID,
	}
}

func wrapInternalError(err error, code int, r *http.Request) error {
	return &spruceError{
		DeveloperError: err.Error(),
		UserError:      genericUserErrorMessage,
		RequestID:      GetContext(r).RequestID,
		HTTPStatusCode: code,
	}
}

func WriteError(err error, w http.ResponseWriter, r *http.Request) {
	switch err := err.(type) {
	case *spruceError:
		err.RequestID = GetContext(r).RequestID
		writeSpruceError(&spruceError{
			UserError:          err.UserError,
			DeveloperError:     err.DeveloperError,
			DeveloperErrorCode: err.DeveloperErrorCode,
			HTTPStatusCode:     err.HTTPStatusCode,
			RequestID:          GetContext(r).RequestID,
		}, w, r)
	case errors.UserError:
		writeSpruceError(&spruceError{
			UserError:      err.UserError(),
			DeveloperError: err.Error(),
			HTTPStatusCode: http.StatusInternalServerError,
			RequestID:      GetContext(r).RequestID,
		}, w, r)
	default:
		writeSpruceError(wrapInternalError(err, http.StatusInternalServerError, r).(*spruceError), w, r)
	}
}

func WriteValidationError(msg string, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(NewValidationError(msg, r).(*spruceError), w, r)
}

// WriteBadRequestError is used for errors that occur during parsing of the HTTP request.
func WriteBadRequestError(err error, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(wrapInternalError(err, http.StatusBadRequest, r).(*spruceError), w, r)
}

// WriteAccessNotAllowedError is used when the user is authenticated but not
// authorized to access a given resource. Hopefully the user will never see
// this since the client shouldn't present the option to begin with.
func WriteAccessNotAllowedError(w http.ResponseWriter, r *http.Request) {
	writeSpruceError(&spruceError{
		UserError:      "Access not permitted",
		RequestID:      GetContext(r).RequestID,
		HTTPStatusCode: http.StatusForbidden,
	}, w, r)
}

func WriteResourceNotFoundError(msg string, w http.ResponseWriter, r *http.Request) {
	writeSpruceError(&spruceError{
		UserError:      msg,
		RequestID:      GetContext(r).RequestID,
		HTTPStatusCode: http.StatusNotFound,
	}, w, r)
}

func writeSpruceError(err *spruceError, w http.ResponseWriter, r *http.Request) {
	golog.Context(
		"RequestID", err.RequestID,
		"ErrorCode", err.DeveloperErrorCode,
	).Logf(2, golog.ERR, err.DeveloperError)

	// remove the developer error information if we are not dealing with
	// before sending information across the wire
	if !environment.IsDev() {
		err.DeveloperError = ""
	}
	WriteJSONToHTTPResponseWriter(w, err.HTTPStatusCode, err)
}
