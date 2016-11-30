package gql

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

const (
	contactEnumPhone = "PHONE"
	contactEnumEmail = "EMAIL"
)

// contactArgumentsConfig represents the config for arguments referencing a contact
var contactArgumentsConfig = graphql.FieldConfigArgument{
	"id": &graphql.ArgumentConfig{Type: graphql.String},
}

// contactArguments represents arguments for referencing a contact
type contactArguments struct {
	ID string `json:"id"`
}

// parsecontactArguments parses the contact arguments out of requests params
func parseContactArguments(args map[string]interface{}) *contactArguments {
	cArgs := &contactArguments{}
	if args != nil {
		if iid, ok := args["id"]; ok {
			if id, ok := iid.(string); ok {
				cArgs.ID = id
			}
		}
	}
	return cArgs
}

// contactField returns is a graphql field for Querying an contact object
var contactField = &graphql.Field{
	Type:    contactType,
	Args:    contactArgumentsConfig,
	Resolve: contactResolve,
}

func contactResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	args := parseContactArguments(p.Args)
	golog.ContextLogger(ctx).Debugf("Resolving contact with args %+v", args)
	if args.ID == "" {
		return nil, nil
	}
	return getContact(ctx, client.Directory(p), args.ID)
}

// contactEnumType represents the possible enum values mapped to contact types
var contactEnumType = graphql.NewEnum(
	graphql.EnumConfig{
		Name: "ContactType",
		Values: graphql.EnumValueConfigMap{
			contactEnumPhone: &graphql.EnumValueConfig{
				Value: contactEnumPhone,
			},
			contactEnumEmail: &graphql.EnumValueConfig{
				Value: contactEnumEmail,
			},
		},
	},
)

// newContactType returns a type object representing an contact contact
var contactType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ContactInfo",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type":        &graphql.Field{Type: graphql.NewNonNull(contactEnumType)},
			"value":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"provisioned": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"label":       &graphql.Field{Type: graphql.String},
		},
	})

func getContact(ctx context.Context, dirCli directory.DirectoryClient, id string) (*models.Contact, error) {
	resp, err := dirCli.Contact(ctx, &directory.ContactRequest{
		ContactID: id,
	})
	if err != nil {
		golog.ContextLogger(ctx).Warningf("Error while fetching contact %s", err)
		return nil, errors.Trace(err)
	}
	return models.TransformContactToModel(resp.Contact), nil
}
