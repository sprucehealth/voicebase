package main

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	lerrors "github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// profiles
type profileSectionInput struct {
	Title string `gql:"title,nonempty"`
	Body  string `gql:"body"`
}

var profileSectionInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ProfileSectionInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"title":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"body":             &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

// createProfile
type createProfileInput struct {
	ClientMutationID string                 `gql:"clientMutationId"`
	EntityID         string                 `gql:"entityID,nonempty"`
	DisplayName      string                 `gql:"displayName,nonempty"`
	ImageMediaID     string                 `gql:"imageMediaID,nonempty"`
	Sections         []*profileSectionInput `gql:"sections,nonempty"`
}

var createProfileInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateProfileInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"displayName":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"imageMediaID":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"sections":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(profileSectionInputType))},
		},
	},
)

type createProfileOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	UUID             string         `json:"uuid,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Entity           *models.Entity `json:"entity"`
}

const (
	createProfileErrorCodePracticeNameOrFirstAndLastRequired = "PRACTICE_NAME_OR_FIRST_LAST_TITLE_REQUIRED"
	createProfileErrorCodeInvalidMediaID                     = "INVALID_MEDIA_ID"
)

var createProfileErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CreateProfileErrorCode",
	Values: graphql.EnumValueConfigMap{
		createProfileErrorCodeInvalidMediaID: &graphql.EnumValueConfig{
			Value:       createProfileErrorCodeInvalidMediaID,
			Description: "The provided media id is not valid.",
		},
	},
})

var createProfileOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateProfilePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"uuid":             &graphql.Field{Type: graphql.String},
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: createProfileErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"entity":           &graphql.Field{Type: entityType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createProfileOutput)
			return ok
		},
	},
)

// Note/TODO: Create and Update profile share a lot of the same check logic but cannot be resused due to differing return error types
//  Should think about a pattern to allow more reuse
var createProfileMutation = &graphql.Field{
	Type: graphql.NewNonNull(createProfileOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createProfileInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
			svc := serviceFromParams(p)
			ram := raccess.ResourceAccess(p)
			ctx := p.Context

			var in createProfileInput
			if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, errors.InternalError(ctx, err)
			}

			// Check that our media ID is valid
			if _, err := ram.MediaInfo(ctx, in.ImageMediaID); lerrors.Cause(err) == raccess.ErrNotFound {
				return &createProfileOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        createProfileErrorCodeInvalidMediaID,
					ErrorMessage:     "The provided media id is not valid.",
				}, nil
			} else if err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			ent, err := updateEntityProfile(ctx, ram, "", in.EntityID, in.ImageMediaID, in.DisplayName, in.Sections, svc.staticURLPrefix)
			if lerrors.Cause(err) == raccess.ErrNotFound {
				return nil, errors.ErrNotFound(ctx, fmt.Sprintf("Resource for profile creation for %s", in.EntityID))
			} else if err != nil {
				return nil, err
			}

			return &createProfileOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          true,
				Entity:           ent,
			}, nil
		})),
}

// updateProfile
type updateProfileInput struct {
	ClientMutationID string                 `gql:"clientMutationId"`
	ProfileID        string                 `gql:"profileID,nonempty"`
	DisplayName      string                 `gql:"displayName,nonempty"`
	ImageMediaID     string                 `gql:"imageMediaID,nonempty"`
	Sections         []*profileSectionInput `gql:"sections,nonempty"`
}

var updateProfileInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateProfileInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"profileID":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"displayName":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"imageMediaID":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"sections":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(profileSectionInputType))},
		},
	},
)

type updateProfileOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	UUID             string         `json:"uuid,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Entity           *models.Entity `json:"entity"`
}

const (
	updateProfileErrorCodeInvalidMediaID = "INVALID_MEDIA_ID"
)

var updateProfileErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "UpdateProfileErrorCode",
	Values: graphql.EnumValueConfigMap{
		updateProfileErrorCodeInvalidMediaID: &graphql.EnumValueConfig{
			Value:       updateProfileErrorCodeInvalidMediaID,
			Description: "The provided media id is not valid.",
		},
	},
})

var updateProfileOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateProfilePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"uuid":             &graphql.Field{Type: graphql.String},
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: updateProfileErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"entity":           &graphql.Field{Type: entityType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateProfileOutput)
			return ok
		},
	},
)

var updateProfileMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateProfileOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateProfileInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
			svc := serviceFromParams(p)
			ram := raccess.ResourceAccess(p)
			ctx := p.Context

			var in updateProfileInput
			if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, errors.InternalError(ctx, err)
			}

			// Check that our media ID is valid
			if _, err := ram.MediaInfo(ctx, in.ImageMediaID); lerrors.Cause(err) == raccess.ErrNotFound {
				return &updateProfileOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        updateProfileErrorCodeInvalidMediaID,
					ErrorMessage:     "The provided media id is not valid.",
				}, nil
			} else if err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			ent, err := updateEntityProfile(ctx, ram, in.ProfileID, "", in.ImageMediaID, in.DisplayName, in.Sections, svc.staticURLPrefix)
			if lerrors.Cause(err) == raccess.ErrNotFound {
				return nil, errors.ErrNotFound(ctx, fmt.Sprintf("Resource for profile update %s", in.ProfileID))
			} else if err != nil {
				return nil, err
			}

			return &updateProfileOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          true,
				Entity:           ent,
			}, nil
		})),
}

func updateEntityProfile(ctx context.Context, ram raccess.ResourceAccessor, profileID, entityID, imageMediaID, customDisplayName string, psis []*profileSectionInput, staticURLPrefix string) (*models.Entity, error) {
	sections := make([]*directory.ProfileSection, len(psis))
	for i, s := range psis {
		sections[i] = &directory.ProfileSection{
			Title: s.Title,
			Body:  s.Body,
		}
	}

	// Perform the profile create and entity update serially so that we can leverage the authorization of the update call
	profile, err := ram.UpdateProfile(ctx, &directory.UpdateProfileRequest{
		ProfileID: profileID,
		Profile: &directory.Profile{
			EntityID:    entityID,
			DisplayName: customDisplayName,
			Sections:    sections,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return transformEntityToResponse(ctx, staticURLPrefix, profile.Entity, devicectx.SpruceHeaders(ctx), gqlctx.Account(ctx))
}
