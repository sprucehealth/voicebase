package errors

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/graphql/gqlerrors"
	"golang.org/x/net/context"
)

// New returns an error that formats as the given text.
// Copying this over since this package has the same name
// as the stdlib package.
var New = errors.New

// Also expose the commonly used trace functionality from the errors lib package
var Trace = errors.Trace

type ErrorType string

const (
	ErrTypeExpired          ErrorType = "EXPIRED"
	ErrTypeInternal         ErrorType = "INTERNAL"
	ErrTypeNotAuthenticated ErrorType = "NOT_AUTHENTICATED"
	ErrTypeNotAuthorized    ErrorType = "NOT_AUTHORIZED"
	ErrTypeNotFound         ErrorType = "NOT_FOUND"
	ErrTypeUnknown          ErrorType = "UNKNOWN"
	ErrTypeNotSupported     ErrorType = "NOT_SUPPORTED"
)

// ErrNotAuthenticated returns the standard not authenticated user error
func ErrNotAuthenticated(ctx context.Context) error {
	return UserError(ctx, ErrTypeNotAuthenticated, "Please sign in to continue.")
}

// ErrNotAuthorized returns the standard not authorized user error
func ErrNotAuthorized(ctx context.Context, resourceID string) error {
	acc := gqlctx.Account(ctx)
	rid := gqlctx.RequestID(ctx)
	golog.LogDepthf(1, golog.WARN, "NotAuthorized: Account %+v attempted to access resource %s and is not authorized [RequestID %d]", acc, resourceID, rid)
	return UserError(ctx, ErrTypeNotAuthorized, "This account is not authorized to access the requested resource.")
}

// ErrNotFound returns the standard not found error
func ErrNotFound(ctx context.Context, resourceID string) error {
	acc := gqlctx.Account(ctx)
	rid := gqlctx.RequestID(ctx)
	golog.LogDepthf(1, golog.WARN, "NotFound: Account %+v requested resource %s and it was not found [RequestID %d]", acc, resourceID, rid)
	return UserError(ctx, ErrTypeNotFound, "This requested resource could not be found.")
}

// UserError created a message with user facing content
func UserError(ctx context.Context, typ ErrorType, m string, a ...interface{}) error {
	return gqlerrors.FormattedError{
		Message:     fmt.Sprintf("Request %d", gqlctx.RequestID(ctx)),
		Type:        string(typ),
		UserMessage: fmt.Sprintf(m, a...),
	}
}

// InternalError logs the provided internal error and returns a sanitized
// versions since we don't want internal details leaking over graphql errors.
func InternalError(ctx context.Context, err error) error {
	rid := gqlctx.RequestID(ctx)
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
