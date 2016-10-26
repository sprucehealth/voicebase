package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestThreadItemUpdate(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.ThreadItem, "ti_2").WithReturns(
		&threading.ThreadItem{
			ID:      "ti_2",
			Deleted: false,
			Item: &threading.ThreadItem_MessageUpdate{
				MessageUpdate: &threading.MessageUpdate{
					ThreadItemID: "ti_1",
					Message: &threading.Message{
						Text: "Foo",
					},
				},
			},
		}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		query _ {
			node(id: "ti_2") {
				... on ThreadItem {
					id
					data {
						... on MessageUpdate {
							threadItem {
								id
							}
						}
					}
				}
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"node": {
				"id": "ti_2",
				"data": {
					"threadItem": {
						"id": "ti_1"
					}
				}
			}
		}
	}`, res)
}

func TestThreadItemDelete(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.ThreadItem, "ti_2").WithReturns(
		&threading.ThreadItem{
			ID:      "ti_2",
			Deleted: false,
			Item: &threading.ThreadItem_MessageDelete{
				MessageDelete: &threading.MessageDelete{
					ThreadItemID: "ti_1",
				},
			},
		}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		query _ {
			node(id: "ti_2") {
				... on ThreadItem {
					id
					data {
						... on MessageDelete {
							threadItem {
								id
							}
						}
					}
				}
			}
		}`, nil)
	responseEquals(t, `{
		"data": {
			"node": {
				"id": "ti_2",
				"data": {
					"threadItem": {
						"id": "ti_1"
					}
				}
			}
		}
	}`, res)
}
