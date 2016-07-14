package authentication

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/libs/auth"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// authenticate

const (
	authenticateErrorBadLogin = "BAD_LOGIN"
)

var authenticateErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "AuthenticateErrorCode",
	Description: "Result of authenticate mutation",
	Values: graphql.EnumValueConfigMap{
		authenticateErrorBadLogin: &graphql.EnumValueConfig{
			Value:       authenticateErrorBadLogin,
			Description: "The login failed",
		},
	},
})

type authenticateOutput struct {
	Success      bool   `json:"success"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	ID           string `json:"id"`
}

var authenticateOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AuthenticatePayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":    &graphql.Field{Type: authenticateErrorCodeEnum},
			"errorMessage": &graphql.Field{Type: graphql.String},
			"username":     &graphql.Field{Type: graphql.String},
			"id":           &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*authenticateOutput)
			return ok
		},
	},
)

type authenticateInput struct {
	Username string `gql:"username,nonempty"`
	Password string `gql:"password,nonempty"`
}

var authenticateInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AuthenticateInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"username": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"password": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

// NewAuthenticateMutation returns the mutation responsible for authenticating with the baymax admin system
func NewAuthenticateMutation(ap auth.AuthenticationProvider) *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(authenticateOutputType),
		Args: graphql.FieldConfigArgument{
			common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(authenticateInputType)},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var in authenticateInput
			if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, fmt.Errorf("TODO: libify InternalError from baymaxgraphql")
			}
			id, err := ap.Authenticate(in.Username, in.Password)
			if err != nil {
				return &authenticateOutput{
					ErrorCode:    authenticateErrorBadLogin,
					ErrorMessage: "Login information is incorrect.",
				}, nil
			}
			return &authenticateOutput{
				Success: true,
				ID:      id,
			}, nil
		},
	}
}
