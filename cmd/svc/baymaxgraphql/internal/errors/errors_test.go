package errors

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/trace/tracectx"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestInternalError(t *testing.T) {
	ctx := context.Background()
	rid := uint64(1234)
	query := "queryString"
	ctx = tracectx.WithRequestID(ctx, rid)
	ctx = gqlctx.WithQuery(ctx, query)

	var entry *golog.Entry
	h := golog.Default().SetHandler(golog.HandlerFunc(func(e *golog.Entry) error {
		entry = e
		return nil
	}))
	defer func() {
		golog.Default().SetHandler(h)
	}()
	InternalError(ctx, New("test"))
	ft := golog.LogfmtFormatter()
	entry.Time = time.Unix(1e9, 0).UTC()
	s := string(ft.Format(entry))
	test.Equals(t, "t=2001-09-09T01:46:40+0000 lvl=ERR msg=\"InternalError: test\" src=errors/errors_test.go:29 requestID=1234 query=queryString\n", s)
}
