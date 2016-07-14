package mutation

import (
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/mutation/authentication"
	"github.com/sprucehealth/backend/libs/auth"
	"github.com/sprucehealth/graphql"
)

// NewRoot returns the root mutation object
func NewRoot(ap auth.AuthenticationProvider) *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Mutation",
			Fields: graphql.Fields{
				"authenticate": authentication.NewAuthenticateMutation(ap),
			},
		})
}
