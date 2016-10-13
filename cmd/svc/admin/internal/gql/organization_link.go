package gql

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func getOrganizationLink(ctx context.Context, inviteCli invite.InviteClient, inviteAPIDomain, id string) (interface{}, error) {
	inv, err := inviteCli.LookupInvite(ctx, &invite.LookupInviteRequest{
		LookupKeyType: invite.LookupInviteRequest_ORGANIZATION_ENTITY_ID,
		LookupKeyOneof: &invite.LookupInviteRequest_OrganizationEntityID{
			OrganizationEntityID: id,
		},
	})
	if grpc.Code(err) == codes.NotFound {
		return "", nil
	} else if err != nil {
		return nil, errors.Errorf("Error while getting org code: %s", err)
	}
	return invite.OrganizationInviteURL(inviteAPIDomain, inv.GetOrganization().Token), nil
}

// createOrganizationLinkInput
type createOrganizationLinkInput struct {
	OrganizationID string `gql:"organizationID"`
}

var createOrganizationLinkInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateOrganizationLinkInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"organizationID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

type createOrganizationLinkOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	OrgLink      string `json:"orgLink"`
}

var createOrganizationLinkOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateOrganizationLinkPayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
			"orgLink":      &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createOrganizationLinkOutput)
			return ok
		},
	},
)

func newCreateOrganizationLinkField() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(createOrganizationLinkOutputType),
		Args: graphql.FieldConfigArgument{
			common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(createOrganizationLinkInputType)},
		},
		Resolve: createOrganizationLinkResolve,
	}
}

func createOrganizationLinkResolve(p graphql.ResolveParams) (interface{}, error) {
	var in createOrganizationLinkInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	golog.ContextLogger(p.Context).Debugf("Creating organization link for %s", in.OrganizationID)
	orgLink, err := createOrganizationLink(p.Context, client.Settings(p), client.Invite(p), client.Domains(p).InviteAPI, in.OrganizationID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if _, err := client.Settings(p).SetValue(p.Context, &settings.SetValueRequest{
		NodeID: in.OrganizationID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key: invite.ConfigKeyOrganizationCode,
			},
			Type: settings.ConfigType_BOOLEAN,
			Value: &settings.Value_Boolean{
				Boolean: &settings.BooleanValue{Value: true},
			},
		},
	}); err != nil {
		return nil, errors.Trace(err)
	}

	golog.ContextLogger(p.Context).Debugf("Created organization link %s", orgLink)
	return &createOrganizationLinkOutput{
		Success: true,
		OrgLink: orgLink,
	}, nil
}

func createOrganizationLink(ctx context.Context, settingsCli settings.SettingsClient, inviteCli invite.InviteClient, inviteAPIDomain, orgID string) (string, error) {
	inv, err := inviteCli.LookupInvite(ctx, &invite.LookupInviteRequest{
		LookupKeyType: invite.LookupInviteRequest_ORGANIZATION_ENTITY_ID,
		LookupKeyOneof: &invite.LookupInviteRequest_OrganizationEntityID{
			OrganizationEntityID: orgID,
		},
	})
	if grpc.Code(err) == codes.NotFound {
		resp, err := inviteCli.CreateOrganizationInvite(ctx, &invite.CreateOrganizationInviteRequest{
			OrganizationEntityID: orgID,
		})
		if err != nil {
			return "", errors.Errorf("Error while creating org code for organization %s: %s", orgID, err)
		}
		return invite.OrganizationInviteURL(inviteAPIDomain, resp.Organization.Token), nil
	} else if err != nil {
		return "", errors.Errorf("Error while getting org code: %s", err)
	}
	return invite.OrganizationInviteURL(inviteAPIDomain, inv.GetOrganization().Token), nil
}
