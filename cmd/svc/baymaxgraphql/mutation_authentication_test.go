package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

func TestAuthenticateMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	var acc *auth.Account
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{
		Platform: device.Android,
	})

	email := "someone@example.com"
	password := "toomanysecrets"

	g.ra.Expect(mock.NewExpectation(g.ra.AuthenticateLogin, email, password).WithReturns(&auth.AuthenticateLoginResponse{
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
	acc2 := gqlctx.Account(ctx)
	test.AssertNotNil(t, acc2)
	test.Equals(t, "acc", acc2.ID)
}

func TestUnauthenticateMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{ID: "a_1"}
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = gqlctx.WithAuthToken(ctx, "token")
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{DeviceID: "deviceID"})

	g.ra.Expect(mock.NewExpectation(g.ra.Unauthenticate, "token").WithReturns(&auth.UnauthenticateResponse{}, nil))
	g.notificationC.Expect(mock.NewExpectation(g.notificationC.DeregisterDeviceForPush, "deviceID"))

	res := g.query(ctx, `
		mutation _ {
			unauthenticate(input: {clientMutationId: "a1b2c3"}) {
				clientMutationId
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"unauthenticate": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))

	// Make sure account gets updated in the context
	// acc2 := account(ctx)
	// test.AssertNotNil(t, acc2)
	// test.Equals(t, "acc", acc2.ID)
}

func TestUnauthenticateMutationNoHeaders(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{ID: "a_1"}
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = gqlctx.WithAuthToken(ctx, "token")

	g.ra.Expect(mock.NewExpectation(g.ra.Unauthenticate, "token").WithReturns(&auth.UnauthenticateResponse{}, nil))

	res := g.query(ctx, `
		mutation _ {
			unauthenticate(input: {clientMutationId: "a1b2c3"}) {
				clientMutationId
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"unauthenticate": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))
}

func TestUnauthenticateMutationNoDeviceID(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{ID: "a_1"}
	ctx = gqlctx.WithAccount(ctx, acc)
	ctx = gqlctx.WithAuthToken(ctx, "token")
	ctx = devicectx.WithSpruceHeaders(ctx, &device.SpruceHeaders{DeviceID: ""})

	g.ra.Expect(mock.NewExpectation(g.ra.Unauthenticate, "token").WithReturns(&auth.UnauthenticateResponse{}, nil))

	res := g.query(ctx, `
		mutation _ {
			unauthenticate(input: {clientMutationId: "a1b2c3"}) {
				clientMutationId
				success
			}
		}`, nil)
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"unauthenticate": {
			"clientMutationId": "a1b2c3",
			"success": true
		}
	}
}`, string(b))

	// Make sure account gets updated in the context
	// acc2 := account(ctx)
	// test.AssertNotNil(t, acc2)
	// test.Equals(t, "acc", acc2.ID)
}
