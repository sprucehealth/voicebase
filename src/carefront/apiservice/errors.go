package apiservice

import (
	"carefront/libs/golog"
	"fmt"
	"net/http"
)

type spruceError struct {
	DeveloperError     string `json:"developer_error,omitempty"`
	UserError          string `json:"user_error,omitempty"`
	DeveloperErrorCode int64  `json:"developer_code,string,omitempty"`
	HTTPStatusCode     int    `json:"-"`
	RequestID          int64  `json:"request_id,string,omitempty"`
}

func NewValidationError(msg string, r *http.Request) spruceError {
	return spruceError{
		UserError:      msg,
		HTTPStatusCode: http.StatusBadRequest,
		RequestID:      GetContext(r).RequestID,
	}
}

func wrapInternalError(err error, r *http.Request) spruceError {
	return spruceError{
		DeveloperError: err.Error(),
		UserError:      genericUserErrorMessage,
		RequestID:      GetContext(r).RequestID,
		HTTPStatusCode: http.StatusInternalServerError,
	}
}

func (s spruceError) Error() string {
	var msg string
	if s.DeveloperErrorCode > 0 {
		msg = fmt.Sprintf("RequestID: %d, Error: %s, ErrorCode: %d", s.RequestID, s.DeveloperError, s.DeveloperErrorCode)
	} else {
		msg = fmt.Sprintf("RequestID: %d, Error: %s", s.RequestID, s.DeveloperError)
	}
	return msg
}

var IsDev = false

func WriteError(err error, w http.ResponseWriter, r *http.Request) {
	switch err := err.(type) {
	case spruceError:
		writeSpruceError(err, w, r)
	default:
		writeSpruceError(wrapInternalError(err, r), w, r)
	}
}

func WriteValidationError(msg string, w http.ResponseWriter, r *http.Request) {
	WriteError(NewValidationError(msg, r), w, r)
}

func writeSpruceError(err spruceError, w http.ResponseWriter, r *http.Request) {
	golog.Logf(3, golog.ERR, err.Error())

	// remove the developer error information if we are not dealing with
	// before sending information across the wire
	if !IsDev {
		err.DeveloperError = ""
	}
	WriteJSONToHTTPResponseWriter(w, err.HTTPStatusCode, err)
}
