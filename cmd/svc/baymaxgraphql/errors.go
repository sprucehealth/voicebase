package main

import (
	"fmt"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/graphql/gqlerrors"
	"golang.org/x/net/context"
)

type errorType string

const (
	errTypeExpired          errorType = "EXPIRED"
	errTypeInternal         errorType = "INTERNAL"
	errTypeNotAuthenticated errorType = "NOT_AUTHENTICATED"
	errTypeNotAuthorized    errorType = "NOT_AUTHORIZED"
	errTypeNotFound         errorType = "NOT_FOUND"
)

func errNotAuthenticated(ctx context.Context) error {
	return userError(ctx, errTypeNotAuthenticated, "Please sign in to continue.")
}

func userError(ctx context.Context, typ errorType, m string, a ...interface{}) error {
	return gqlerrors.FormattedError{
		Message:     fmt.Sprintf("Request %d", requestIDFromContext(ctx)),
		Type:        string(typ),
		UserMessage: fmt.Sprintf(m, a...),
	}
}

// internalError logs the provided internal error and returns a sanitized
// versions since we don't want internal details leaking over graphql errors.
func internalError(ctx context.Context, err error) error {
	rid := requestIDFromContext(ctx)
	golog.LogDepthf(1, golog.ERR, "%s [RequestID %d]", err, rid)
	userMessage := "Something went wrong on the server."
	if !environment.IsProd() {
		return gqlerrors.FormattedError{
			Message:     fmt.Sprintf("Internal error [%d]: %s", rid, err),
			Type:        string(errTypeInternal),
			UserMessage: userMessage,
		}
	}
	return gqlerrors.FormattedError{
		Message:     fmt.Sprintf("Internal error [%d]", rid),
		Type:        string(errTypeInternal),
		UserMessage: userMessage,
	}
}
