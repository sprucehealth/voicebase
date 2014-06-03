package apiservice

import (
	"carefront/libs/golog"
	"encoding/json"
	"net/http"
)

type SpruceError struct {
	DeveloperError     string `json:"developer_error,omitempty"`
	UserError          string `json:"user_error,omitempty"`
	DeveloperErrorCode int64  `json:"developer_code,string,omitempty"`
	HTTPStatusCode     int    `json:"-"`
	RequestID          int64  `json:"request_id,string,omitempty"`
}

func NewValidationError(msg string) SpruceError {
	return SpruceError{
		UserError:      msg,
		HTTPStatusCode: http.StatusBadRequest,
	}
}

func wrapInternalError(err error, r *http.Request) SpruceError {
	return SpruceError{
		DeveloperError: err.Error(),
		UserError:      genericUserErrorMessage,
		RequestID:      GetContext(r).RequestID,
	}
}

func (s SpruceError) Error() string {
	jsonData, err := json.Marshal(&s)
	if err != nil {
		return s.DeveloperError
	} else {
		return string(jsonData)
	}
}

var IsDev = false

func WriteError(err error, w http.ResponseWriter, r *http.Request) {
	switch err := err.(type) {
	case SpruceError:
		err.RequestID = GetContext(r).RequestID
		golog.Logf(2, golog.ERR, err.Error())

		// remove the developer error information if we are not dealing with
		// before sending information across the wire
		if !IsDev {
			err.DeveloperError = ""
		}
		WriteJSONToHTTPResponseWriter(w, err.HTTPStatusCode, err)
	default:
		WriteJSONToHTTPResponseWriter(w, http.StatusInternalServerError, wrapInternalError(err, r))
	}
}
