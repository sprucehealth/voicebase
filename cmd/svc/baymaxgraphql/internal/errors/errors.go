package errors

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// Expose functionality from the errors pkg for convenience
var (
	New    = errors.New
	Trace  = errors.Trace
	Errorf = errors.Errorf
)

type ErrorType string

const (
	ErrTypeExpired          ErrorType = "EXPIRED"
	ErrTypeInternal         ErrorType = "INTERNAL"
	ErrTypeNotAuthenticated ErrorType = "NOT_AUTHENTICATED"
	ErrTypeNotAuthorized    ErrorType = "NOT_AUTHORIZED"
	ErrTypeNotFound         ErrorType = "NOT_FOUND"
	ErrTypeUnknown          ErrorType = "UNKNOWN"
	ErrTypeNotSupported     ErrorType = "NOT_SUPPORTED"
	ErrTypeInputError       ErrorType = "INPUT_ERROR"
)

// ErrNotAuthenticated returns the standard not authenticated user error
func ErrNotAuthenticated(ctx context.Context) error {
	return UserError(ctx, ErrTypeNotAuthenticated, "Please sign in to continue.")
}

// ErrNotAuthorized returns the standard not authorized user error
func ErrNotAuthorized(ctx context.Context, resourceID string) error {
	acc := gqlctx.Account(ctx)
	rid := httputil.RequestID(ctx)
	//Fuzz identifiables before logging att the err level
	golog.LogDepthf(1, golog.INFO, "NotAuthorized: Account %+v attempted to access resource %s and is not authorized [RequestID %d]", auth.ObfuscateAccount(acc), resourceID, rid)
	return UserError(ctx, ErrTypeNotAuthorized, "This account is not authorized to access the requested resource.")
}

// ErrNotSupported returns the standard not not supported user error
func ErrNotSupported(ctx context.Context, err error) error {
	acc := gqlctx.Account(ctx)
	rid := httputil.RequestID(ctx)
	golog.LogDepthf(1, golog.WARN, "Account %+v NotSupported: %s [Request: %d]", acc, err.Error(), rid)
	return UserError(ctx, ErrTypeNotSupported, "This functionality is not supported.")
}

// ErrNotFound returns the standard not found error
func ErrNotFound(ctx context.Context, resourceID string) error {
	acc := gqlctx.Account(ctx)
	rid := httputil.RequestID(ctx)
	golog.LogDepthf(1, golog.WARN, "NotFound: Account %+v requested resource %s and it was not found [RequestID %d]", acc, resourceID, rid)
	return UserError(ctx, ErrTypeNotFound, "This requested resource could not be found.")
}

// UserError created a message with user facing content
func UserError(ctx context.Context, typ ErrorType, m string, a ...interface{}) error {
	return gqlerrors.FormattedError{
		Message:     fmt.Sprintf("Request %d", httputil.RequestID(ctx)),
		Type:        string(typ),
		UserMessage: fmt.Sprintf(m, a...),
	}
}

// InternalError logs the provided internal error and returns a sanitized
// versions since we don't want internal details leaking over graphql errors.
func InternalError(ctx context.Context, err error) error {
	rid := httputil.RequestID(ctx)
	if Type(err) != ErrTypeUnknown {
		golog.LogDepthf(1, golog.WARN,
			"Well Formed InternalError: The following error was well formed but still logged as Internal. Omitting internal wrapper: %s [RequestID %d]", err, rid)
		return err
	}
	golog.Context("requestID", rid, "query", gqlctx.Query(ctx)).LogDepthf(1, golog.ERR, "InternalError: %s", err)
	userMessage := "Something went wrong on the server."
	if !environment.IsProd() {
		return gqlerrors.FormattedError{
			Message:     fmt.Sprintf("Internal error [%d]: %s", rid, err),
			Type:        string(ErrTypeInternal),
			UserMessage: userMessage,
		}
	}
	return gqlerrors.FormattedError{
		Message:     fmt.Sprintf("Internal error [%d]", rid),
		Type:        string(ErrTypeInternal),
		UserMessage: userMessage,
	}
}

// Type returns the underlying error type and UNKNOWN if one cannot be found
func Type(err error) ErrorType {
	if err == nil {
		return ErrTypeUnknown
	}
	ferr, ok := err.(gqlerrors.FormattedError)
	if !ok {
		return ErrTypeUnknown
	}
	return ErrorType(ferr.Type)
}
