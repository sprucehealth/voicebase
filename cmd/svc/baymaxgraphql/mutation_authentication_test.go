package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestAuthenticateMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *account
	ctx = ctxWithAccount(ctx, acc)

	email := "someone@example.com"
	password := "toomanysecrets"

	g.authC.Expect(mock.NewExpectation(g.authC.AuthenticateLogin, &auth.AuthenticateLoginRequest{
		Email:    email,
		Password: password,
	}).WithReturns(&auth.AuthenticateLoginResponse{
		Token: &auth.AuthToken{
			Value: "token",
		},
		Account: &auth.Account{
			ID: "acc",
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($email: String!, $password: String!) {
			authenticate(input: {
				clientMutationId: "a1b2c3",
				email: $email,
				password: $password,
			}) {
				clientMutationId
				success
			}
		}`, map[string]interface{}{
		"email":    email,
		"password": password,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"authenticate": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))

	// Make sure account gets updated in the context
	acc2 := accountFromContext(ctx)
	test.AssertNotNil(t, acc2)
	test.Equals(t, "acc", acc2.ID)
}
