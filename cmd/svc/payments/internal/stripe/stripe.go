package stripe

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	"context"

	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/client"
)

// DashboardURL returns the appropriate URL for the dashboard based on Live/Test
// environments
func DashboardURL() string {
	if environment.IsProd() {
		return "https://dashboard.stripe.com"
	}
	return "https://dashboard.stripe.com/test"
}

// IdempotentStripeClient exposes a wrapped stripe client with built in idempotency mechanisms
type IdempotentStripeClient interface {
	Account(ctx context.Context) (*stripe.Account, error)
	Card(ctx context.Context, id string, cParams *stripe.CardParams) (*stripe.Card, error)
	CreateCard(ctx context.Context, cParams *stripe.CardParams, opts ...CallOption) (*stripe.Card, error)
	CreateCharge(ctx context.Context, cParams *stripe.ChargeParams, opts ...CallOption) (*stripe.Charge, error)
	CreateCustomer(ctx context.Context, cParams *stripe.CustomerParams, opts ...CallOption) (*stripe.Customer, error)
	DeleteCard(ctx context.Context, id string, cParams *stripe.CardParams, opts ...CallOption) error
	Token(ctx context.Context, tParams *stripe.TokenParams, opts ...CallOption) (*stripe.Token, error)
}

// CallOption represents an option available to a stripe call
type CallOption int

const (
	// PreserveIdempotencyKey indicates that the provided idempotency key should be preserved
	PreserveIdempotencyKey CallOption = 1 << iota
)

type callOptions []CallOption

func (cos callOptions) Has(opt CallOption) bool {
	for _, o := range cos {
		if o == opt {
			return true
		}
	}
	return false
}

type stripeWrapper struct {
	accessToken  string
	stripeClient *client.API
}

// NewClient returns a new stripe client using the provided credentials
func NewClient(accessToken string) IdempotentStripeClient {
	sc := &client.API{}
	sc.Init(accessToken, nil)
	return &stripeWrapper{
		accessToken:  accessToken,
		stripeClient: sc,
	}
}

func (sw *stripeWrapper) Account(ctx context.Context) (*stripe.Account, error) {
	account, err := sw.stripeClient.Account.Get()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return account, nil
}

func (sw *stripeWrapper) Card(ctx context.Context, id string, cParams *stripe.CardParams) (*stripe.Card, error) {
	card, err := sw.stripeClient.Cards.Get(id, cParams)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return card, nil
}

func (sw *stripeWrapper) CreateCard(ctx context.Context, cParams *stripe.CardParams, opts ...CallOption) (*stripe.Card, error) {
	if !callOptions(opts).Has(PreserveIdempotencyKey) {
		idk, err := sw.idempotencyKey("create_card", cParams)
		if err != nil {
			return nil, errors.Trace(err)
		}
		cParams.Params.IdempotencyKey = idk
	}
	card, err := sw.stripeClient.Cards.New(cParams)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return card, nil
}

func (sw *stripeWrapper) CreateCharge(ctx context.Context, cParams *stripe.ChargeParams, opts ...CallOption) (*stripe.Charge, error) {
	if !callOptions(opts).Has(PreserveIdempotencyKey) {
		idk, err := sw.idempotencyKey("create_charge", cParams)
		if err != nil {
			return nil, errors.Trace(err)
		}
		cParams.Params.IdempotencyKey = idk
	}
	card, err := sw.stripeClient.Charges.New(cParams)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return card, nil
}

func (sw *stripeWrapper) DeleteCard(ctx context.Context, id string, cParams *stripe.CardParams, opts ...CallOption) error {
	if !callOptions(opts).Has(PreserveIdempotencyKey) {
		idk, err := sw.idempotencyKey("delete_card", cParams)
		if err != nil {
			return errors.Trace(err)
		}
		cParams.Params.IdempotencyKey = idk
	}
	if _, err := sw.stripeClient.Cards.Del(id, cParams); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (sw *stripeWrapper) CreateCustomer(ctx context.Context, cParams *stripe.CustomerParams, opts ...CallOption) (*stripe.Customer, error) {
	if !callOptions(opts).Has(PreserveIdempotencyKey) {
		idk, err := sw.idempotencyKey("create_customer", cParams)
		if err != nil {
			return nil, errors.Trace(err)
		}
		cParams.Params.IdempotencyKey = idk
	}
	customer, err := sw.stripeClient.Customers.New(cParams)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return customer, nil
}

func (sw *stripeWrapper) Token(ctx context.Context, tParams *stripe.TokenParams, opts ...CallOption) (*stripe.Token, error) {
	if !callOptions(opts).Has(PreserveIdempotencyKey) {
		idk, err := sw.idempotencyKey("token", tParams)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tParams.Params.IdempotencyKey = idk
	}
	token, err := sw.stripeClient.Tokens.New(tParams)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return token, nil
}

// ErrType converts a generic error into a well formed stripe api error type
func ErrType(err error) stripe.ErrorType {
	if err == nil {
		return stripe.ErrorType("")
	}
	if stripeErr, ok := err.(*stripe.Error); ok {
		return stripeErr.Type
	}
	return stripe.ErrorType("")
}

// ErrCode converts a generic error into a well formed stripe api error code
func ErrCode(err error) stripe.ErrorCode {
	if err == nil {
		return stripe.ErrorCode("")
	}
	if stripeErr, ok := err.(*stripe.Error); ok {
		return stripeErr.Code
	}
	return stripe.ErrorCode("")
}

// ErrMessage converts a generic error into a well formed stripe api error message
func ErrMessage(err error) string {
	if err == nil {
		return ""
	}
	if stripeErr, ok := err.(*stripe.Error); ok {
		return stripeErr.Msg
	}
	return ""
}

func (sw *stripeWrapper) idempotencyKey(apiName string, req interface{}) (string, error) {
	// TODO: Is json.Marshal deterministic?
	bIdemSource, err := json.Marshal(req)
	if err != nil {
		return "", errors.Trace(err)
	}

	// suffix the serialized info with the access token and impersonation account
	bIdemSource = append(bIdemSource, []byte(apiName+sw.accessToken)...)

	// TODO: Is MD5 right for this? Is this actually deterministic?
	// use the MD5 of our request to create a unique deterministic idempotency key
	idk := string(fmt.Sprintf("%x", md5.Sum(bIdemSource)))
	golog.Debugf("Stripe Idempotency - Call Name: %s - Key Source: %s - Key: %s", apiName, string(bIdemSource), idk)
	return idk, nil
}
