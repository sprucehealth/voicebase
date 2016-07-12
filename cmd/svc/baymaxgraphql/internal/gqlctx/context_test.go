package gqlctx

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/auth"
)

func TestQuery(t *testing.T) {
	test.Equals(t, "query", Query(WithQuery(context.Background(), "query")))
}

func TestAuthToken(t *testing.T) {
	test.Equals(t, "token", AuthToken(WithAuthToken(context.Background(), "token")))
}

func TestClientEncryptionKey(t *testing.T) {
	test.Equals(t, "key", ClientEncryptionKey(WithClientEncryptionKey(context.Background(), "key")))
}

func TestAccount(t *testing.T) {
	ctx := context.Background()
	ctx = WithAccount(ctx, nil)
	test.Equals(t, (*auth.Account)(nil), Account(ctx))

	acc := &auth.Account{ID: "123"}
	InPlaceWithAccount(ctx, acc)
	test.Equals(t, acc, Account(ctx))
}
