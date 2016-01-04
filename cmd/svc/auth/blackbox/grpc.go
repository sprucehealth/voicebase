package blackbox

import (
	"time"

	"github.com/sprucehealth/backend/cmd/svc/blackbox/harness"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Tests contains the test methods for the auth service
type tests struct{}

// NewTests returns an initialized instance of tests
func NewTests() harness.TestSuite {
	return &tests{}
}

// SuiteName returns the name of this test suite
func (t *tests) SuiteName() string {
	return "AuthService"
}

// GeneratePayload conforms to the BBTest harness payload generation
func (t *tests) GeneratePayload() interface{} {
	conn, err := grpc.Dial(harness.GetConfig("auth_service_endpoint"), grpc.WithInsecure(), grpc.WithTimeout(2*time.Second))
	if err != nil {
		golog.Fatalf("Unable to dial grpc server: %s", err)
	}
	client := auth.NewAuthClient(conn)
	return client
}

var (
	maxAccountFirstNameSize          int64 = 150
	maxAccountLastNameSize           int64 = 150
	maxAccountPasswordLength         int64 = 250
	maxAuthTokenAttributes           int64 = 5
	maxAuthTokenAttributeKeyLength   int64 = 20
	maxAuthTokenAttributeValueLength int64 = 20
)

func optionalTokenAttributes() map[string]string {
	// token attributes is optional
	var tokenAttributes map[string]string
	if harness.RandBool() {
		tokenAttributes = make(map[string]string)
		for i := int64(0); i < harness.RandInt64N(maxAuthTokenAttributes); i++ {
			tokenAttributes[harness.RandLengthString(maxAuthTokenAttributeKeyLength)] = harness.RandLengthString(maxAuthTokenAttributeValueLength)
		}
	}
	return tokenAttributes
}

func randomValidCreateAccountRequest() *auth.CreateAccountRequest {
	return &auth.CreateAccountRequest{
		FirstName:       harness.RandLengthString(maxAccountFirstNameSize),
		LastName:        harness.RandLengthString(maxAccountLastNameSize),
		Email:           harness.RandEmail(),
		PhoneNumber:     harness.RandPhoneNumber(),
		Password:        harness.RandLengthString(maxAccountPasswordLength),
		TokenAttributes: optionalTokenAttributes(),
	}
}

// TODO: Deal with retries related to rand phone number duplication
func createRandomValidAccount(client auth.AuthClient) (*auth.CreateAccountRequest, *auth.CreateAccountResponse) {
	var err error
	var resp *auth.CreateAccountResponse
	req := randomValidCreateAccountRequest()
	golog.Debugf("CreateAccount call: %+v", req)
	harness.Profile("AuthService:CreateAccount", func() { resp, err = client.CreateAccount(context.Background(), req) })
	golog.Debugf("CreateAccount response: %+v", resp)
	harness.FailErr(err)
	harness.AssertNotNil(resp)
	harness.Assert(resp.Success, resp)
	harness.AssertNil(resp.Failure, resp)
	assertValidAuthToken(resp.Token)
	assertValidAccount(resp.Account)
	return req, resp
}

func assertValidAuthToken(token *auth.AuthToken) {
	harness.AssertNotNil(token)
	harness.Assert(token.Value != "")
	harness.Assert(token.ExpirationEpoch != 0)
}

func assertValidAccount(account *auth.Account) {
	harness.AssertNotNil(account)
	harness.Assert(account.ID != "")
	harness.Assert(account.FirstName != "")
	harness.Assert(account.LastName != "")
}

func checkAuthentication(client auth.AuthClient, token string, attributes map[string]string) (*auth.CheckAuthenticationRequest, *auth.CheckAuthenticationResponse) {
	var err error
	var resp *auth.CheckAuthenticationResponse
	req := &auth.CheckAuthenticationRequest{
		Token:           token,
		TokenAttributes: attributes,
	}
	golog.Debugf("CheckAuthentication call: %+v", req)
	harness.Profile("AuthService:CheckAuthentication", func() { resp, err = client.CheckAuthentication(context.Background(), req) })
	golog.Debugf("CheckAuthentication response: %+v", resp)
	harness.FailErr(err)
	harness.Assert(resp.Success, resp)
	harness.AssertNil(resp.Failure, resp)
	if resp.IsAuthenticated {
		assertValidAuthToken(resp.Token)
		assertValidAccount(resp.Account)
	} else {
		harness.AssertNil(resp.Token)
		harness.AssertNil(resp.Account)
	}
	return req, resp
}

func authenticateLogin(client auth.AuthClient, email string, password string) (*auth.AuthenticateLoginRequest, *auth.AuthenticateLoginResponse) {
	var err error
	var resp *auth.AuthenticateLoginResponse
	req := &auth.AuthenticateLoginRequest{
		Email:           email,
		Password:        password,
		TokenAttributes: optionalTokenAttributes(),
	}
	golog.Debugf("AuthenticateLogin call: %+v", req)
	harness.Profile("AuthService:AuthenticateLogin", func() { resp, err = client.AuthenticateLogin(context.Background(), req) })
	golog.Debugf("AuthenticateLogin response: %+v", resp)
	harness.FailErr(err)
	harness.Assert(resp.Success, resp)
	harness.AssertNil(resp.Failure, resp)
	assertValidAuthToken(resp.Token)
	assertValidAccount(resp.Account)
	return req, resp
}

func unauthenticate(client auth.AuthClient, token string, tokenAttributes map[string]string) (*auth.UnauthenticateRequest, *auth.UnauthenticateResponse) {
	var err error
	var resp *auth.UnauthenticateResponse
	req := &auth.UnauthenticateRequest{
		Token:           token,
		TokenAttributes: tokenAttributes,
	}
	golog.Debugf("Unauthenticate call: %+v", req)
	harness.Profile("AuthService:AuthenticateLogin", func() { resp, err = client.Unauthenticate(context.Background(), req) })
	golog.Debugf("Unauthenticate response: %+v", resp)
	harness.FailErr(err)
	harness.Assert(resp.Success, resp)
	harness.AssertNil(resp.Failure, resp)
	return req, resp
}

func getAccount(client auth.AuthClient, accountID string) (*auth.GetAccountRequest, *auth.GetAccountResponse) {
	var err error
	var resp *auth.GetAccountResponse
	req := &auth.GetAccountRequest{
		AccountID: accountID,
	}
	golog.Debugf("GetAccount call: %+v", req)
	harness.Profile("AuthService:GetAccount", func() { resp, err = client.GetAccount(context.Background(), req) })
	golog.Debugf("GetAccount response: %+v", resp)
	harness.FailErr(err)
	harness.Assert(resp.Success, resp)
	harness.AssertNil(resp.Failure, resp)
	assertValidAccount(resp.Account)
	return req, resp
}

// BBTestGRPCBasicAccountCreationAndAuthentication tests the CreateAccount grpc endpoint of the auth service and the subsequent auth check flows
func (t *tests) BBTestGRPCBasicAccountCreationAndAuthentication(client interface{}) {
	authClient, ok := client.(auth.AuthClient)
	if !ok {
		harness.Failf("Unable to unpack client: %+v", client)
	}

	// Create an account
	createAccountReq, createAccountResp := createRandomValidAccount(authClient)

	// Check that the token returned from the create account call is respected
	_, checkAuthenticationResp := checkAuthentication(authClient, createAccountResp.Token.Value, createAccountReq.TokenAttributes)
	harness.Assert(checkAuthenticationResp.IsAuthenticated, checkAuthenticationResp)

	// Check that we can authenticate with the account information that we just created
	authenticateLoginReq, authenticateLoginResp := authenticateLogin(authClient, createAccountReq.Email, createAccountReq.Password)

	// Check that the token returned from the login call is respected
	_, checkAuthenticationResp = checkAuthentication(authClient, authenticateLoginResp.Token.Value, authenticateLoginReq.TokenAttributes)
	harness.Assert(checkAuthenticationResp.IsAuthenticated, checkAuthenticationResp)

	// Check that we can tombstone the token
	unauthenticate(authClient, authenticateLoginResp.Token.Value, authenticateLoginReq.TokenAttributes)

	// Check that the token is now rejected
	_, checkAuthenticationResp = checkAuthentication(authClient, authenticateLoginResp.Token.Value, authenticateLoginReq.TokenAttributes)
	harness.Assert(checkAuthenticationResp.IsAuthenticated == false, checkAuthenticationResp)
}

// BBTestGRPCAccountFetching tests the GetAccount grpc endpoint of the auth service
func (t *tests) BBTestGRPCAccountFetching(client interface{}) {
	authClient, ok := client.(auth.AuthClient)
	if !ok {
		harness.Failf("Unable to unpack client: %+v", client)
	}

	// Create an account
	_, createAccountResp := createRandomValidAccount(authClient)

	// Check that we can fetch the account and get back matching information
	_, getAccountResp := getAccount(authClient, createAccountResp.Account.ID)
	harness.AssertEqual(getAccountResp.Account, createAccountResp.Account)
}
