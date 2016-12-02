package gql

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// practiceLinkArgumentsConfig represents the config for arguments referencing a practice link
var practiceLinkArgumentsConfig = graphql.FieldConfigArgument{
	"practiceCode": &graphql.ArgumentConfig{Type: graphql.String},
}

// practiceLinkArguments represents arguments for referencing an practice link
type practiceLinkArguments struct {
	PracticeCode string `json:"practiceCode"`
}

// PracticeLinkArguments parses the practice link arguments out of requests params
func parsePracticeLinkArguments(args map[string]interface{}) *practiceLinkArguments {
	plArgs := &practiceLinkArguments{}
	if args != nil {
		if ipl, ok := args["practiceCode"]; ok {
			if pl, ok := ipl.(string); ok {
				plArgs.PracticeCode = pl
			}
		}
	}
	return plArgs
}

// practiceLinkField returns is a graphql field for Querying an PracticeLink object
var practiceLinkField = &graphql.Field{
	Type:    practiceLinkType,
	Args:    practiceLinkArgumentsConfig,
	Resolve: practiceLinkResolve,
}

// practiceLinkType is a type representing an practice link
var practiceLinkType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PracticeLink",
		Fields: graphql.Fields{
			"organizationID": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"token":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"url":            &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"tags":           &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
	})

func practiceLinkResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	args := parsePracticeLinkArguments(p.Args)
	golog.ContextLogger(ctx).Debugf("Resolving Practice Link with args %+v", args)
	if args.PracticeCode == "" {
		return nil, nil
	}
	return getPracticeLink(ctx, client.Invite(p), client.Domains(p).InviteAPI, args.PracticeCode)
}

func getPracticeLink(ctx context.Context, inviteCli invite.InviteClient, inviteAPIDomain, token string) (*models.PracticeLink, error) {
	resp, err := inviteCli.LookupInvite(ctx, &invite.LookupInviteRequest{
		LookupKeyOneof: &invite.LookupInviteRequest_Token{
			Token: token,
		},
	})
	if err != nil {
		return nil, errors.Errorf("Error while getting practice link: %s", err)
	}
	if resp.Type != invite.LookupInviteResponse_ORGANIZATION_CODE {
		return nil, errors.Errorf("Invite mapped to token %s is not a practice linke. Got: %+v", token, resp.Invite)
	}
	return models.TransformPracticeLinkToModel(ctx, resp.GetOrganization(), inviteAPIDomain), nil
}

func getPracticeLinksForEntity(ctx context.Context, inviteCli invite.InviteClient, inviteAPIDomain, entityID string) ([]*models.PracticeLink, error) {
	resp, err := inviteCli.LookupOrganizationInvites(ctx, &invite.LookupOrganizationInvitesRequest{
		OrganizationEntityID: entityID,
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Errorf("Error while getting practice link: %s", err)
	}
	return models.TransformPracticeLinksToModel(ctx, resp.OrganizationInvites, inviteAPIDomain), nil
}

// TODO: Rename from Org Link to Practice Link in inputs/outputs
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

var createOrganizationLinkField = &graphql.Field{
	Type: graphql.NewNonNull(createOrganizationLinkOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(createOrganizationLinkInputType)},
	},
	Resolve: createOrganizationLinkResolve,
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

	golog.ContextLogger(p.Context).Debugf("Created organization link %s", orgLink)
	return &createOrganizationLinkOutput{
		Success: true,
		OrgLink: orgLink,
	}, nil
}

func createOrganizationLink(ctx context.Context, settingsCli settings.SettingsClient, inviteCli invite.InviteClient, inviteAPIDomain, orgID string) (string, error) {
	// If we're creating an org link then enable the setting
	if _, err := settingsCli.SetValue(ctx, &settings.SetValueRequest{
		NodeID: orgID,
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
		return "", errors.Trace(err)
	}

	resp, err := inviteCli.CreateOrganizationInvite(ctx, &invite.CreateOrganizationInviteRequest{
		OrganizationEntityID: orgID,
	})
	if err != nil {
		return "", errors.Errorf("Error while creating org code for organization %s: %s", orgID, err)
	}
	return invite.OrganizationInviteURL(inviteAPIDomain, resp.Organization.Token), nil
}

// modifyPracticeLink
type modifyPracticeLinkInput struct {
	PracticeCode string   `gql:"practiceCode,nonempty"`
	Tags         []string `gql:"tags"`
}

var modifyPracticeLinkInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ModifyPracticeLinkInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"practiceCode": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"tags":         &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
	},
)

type modifyPracticeLinkOutput struct {
	Success      bool                 `json:"success"`
	ErrorMessage string               `json:"errorMessage,omitempty"`
	PracticeLink *models.PracticeLink `json:"practiceLink"`
}

var modifyPracticeLinkOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ModifyPracticeLinkPayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
			"practiceLink": &graphql.Field{Type: graphql.NewNonNull(practiceLinkType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*modifyPracticeLinkOutput)
			return ok
		},
	},
)

var modifyPracticeLinkField = &graphql.Field{
	Type: graphql.NewNonNull(createOrganizationLinkOutputType),
	Args: graphql.FieldConfigArgument{
		common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(modifyPracticeLinkInputType)},
	},
	Resolve: modifyPracticeLinkResolve,
}

func modifyPracticeLinkResolve(p graphql.ResolveParams) (interface{}, error) {
	var in modifyPracticeLinkInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	golog.ContextLogger(p.Context).Debugf("Modifying practice link for %+v", in)
	practiceLink, err := modifyPracticeLink(p.Context, client.Invite(p), &in, client.Domains(p).InviteAPI)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &modifyPracticeLinkOutput{
		Success:      true,
		PracticeLink: practiceLink,
	}, nil
}

func modifyPracticeLink(ctx context.Context, inviteCli invite.InviteClient, in *modifyPracticeLinkInput, inviteAPIDomain string) (*models.PracticeLink, error) {
	resp, err := inviteCli.ModifyOrganizationInvite(ctx, &invite.ModifyOrganizationInviteRequest{
		Token: in.PracticeCode,
		Tags:  in.Tags,
	})
	if err != nil {
		return nil, errors.Errorf("Error while modifying practice link for  %+v", in)
	}
	return models.TransformPracticeLinkToModel(ctx, resp.OrganizationInvite, inviteAPIDomain), nil
}
