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

// createEntityProfile
type createEntityProfileInput struct {
	ClientMutationID string                 `gql:"clientMutationId"`
	EntityID         string                 `gql:"entityID,nonempty"`
	DisplayName      string                 `gql:"displayName,nonempty"`
	ImageMediaID     string                 `gql:"imageMediaID"`
	Sections         []*profileSectionInput `gql:"sections,nonempty"`
}

var createEntityProfileInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateEntityProfileInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"displayName":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"imageMediaID":     &graphql.InputObjectFieldConfig{Type: graphql.String},
			"sections":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(profileSectionInputType))},
		},
	},
)

type createEntityProfileOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	UUID             string         `json:"uuid,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Entity           *models.Entity `json:"entity"`
}

const (
	profileErrorCodeInvalidMediaID = "INVALID_MEDIA_ID"
)

var createEntityProfileErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CreateEntityProfileErrorCode",
	Values: graphql.EnumValueConfigMap{
		profileErrorCodeInvalidMediaID: &graphql.EnumValueConfig{
			Value:       profileErrorCodeInvalidMediaID,
			Description: "The provided media id is not valid.",
		},
	},
})

var createEntityProfileOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateEntityProfilePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"uuid":             &graphql.Field{Type: graphql.String},
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: createEntityProfileErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"entity":           &graphql.Field{Type: entityType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createEntityProfileOutput)
			return ok
		},
	},
)

// Note/TODO: Create and Update profile share a lot of the same check logic but cannot be resused due to differing return error types
//  Should think about a pattern to allow more reuse
var createEntityProfileMutation = &graphql.Field{
	Type: graphql.NewNonNull(createEntityProfileOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createEntityProfileInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
			svc := serviceFromParams(p)
			ram := raccess.ResourceAccess(p)
			ctx := p.Context

			var in createEntityProfileInput
			if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, errors.InternalError(ctx, err)
			}

			// Check that our media ID is valid
			if in.ImageMediaID != "" {
				if _, err := ram.MediaInfo(ctx, in.ImageMediaID); lerrors.Cause(err) == raccess.ErrNotFound {
					return &createEntityProfileOutput{
						ClientMutationID: in.ClientMutationID,
						Success:          false,
						ErrorCode:        profileErrorCodeInvalidMediaID,
						ErrorMessage:     "The provided media id is not valid.",
					}, nil
				} else if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
			}

			ent, err := updateEntityProfile(ctx, ram, "", in.EntityID, in.ImageMediaID, in.DisplayName, in.Sections, svc.staticURLPrefix)
			if lerrors.Cause(err) == raccess.ErrNotFound {
				return nil, errors.ErrNotFound(ctx, fmt.Sprintf("Resource for profile creation for %s", in.EntityID))
			} else if err != nil {
				return nil, err
			}

			return &createEntityProfileOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          true,
				Entity:           ent,
			}, nil
		})),
}

// createOrganizationProfile
type createOrganizationProfileInput struct {
	ClientMutationID string                 `gql:"clientMutationId"`
	OrganizationID   string                 `gql:"organizationID,nonempty"`
	DisplayName      string                 `gql:"displayName,nonempty"`
	ImageMediaID     string                 `gql:"imageMediaID"`
	Sections         []*profileSectionInput `gql:"sections,nonempty"`
}

var createOrganizationProfileInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreateOrganizationProfileInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"displayName":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"imageMediaID":     &graphql.InputObjectFieldConfig{Type: graphql.String},
			"sections":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(profileSectionInputType))},
		},
	},
)

type createOrganizationProfileOutput struct {
	ClientMutationID string               `json:"clientMutationId,omitempty"`
	UUID             string               `json:"uuid,omitempty"`
	Success          bool                 `json:"success"`
	ErrorCode        string               `json:"errorCode,omitempty"`
	ErrorMessage     string               `json:"errorMessage,omitempty"`
	Organization     *models.Organization `json:"organization"`
}

var createOrganizationProfileErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CreateOrganizationProfileErrorCode",
	Values: graphql.EnumValueConfigMap{
		profileErrorCodeInvalidMediaID: &graphql.EnumValueConfig{
			Value:       profileErrorCodeInvalidMediaID,
			Description: "The provided media id is not valid.",
		},
	},
})

var createOrganizationProfileOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreateOrganizationProfilePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"uuid":             &graphql.Field{Type: graphql.String},
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: createOrganizationProfileErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"organization":     &graphql.Field{Type: organizationType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createOrganizationProfileOutput)
			return ok
		},
	},
)

// Note/TODO: Create and Update profile share a lot of the same check logic but cannot be resused due to differing return error types
//  Should think about a pattern to allow more reuse
var createOrganizationProfileMutation = &graphql.Field{
	Type: graphql.NewNonNull(createOrganizationProfileOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createOrganizationProfileInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
			svc := serviceFromParams(p)
			ram := raccess.ResourceAccess(p)
			ctx := p.Context

			var in createOrganizationProfileInput
			if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, errors.InternalError(ctx, err)
			}

			// Check that our media ID is valid
			if in.ImageMediaID != "" {
				if _, err := ram.MediaInfo(ctx, in.ImageMediaID); lerrors.Cause(err) == raccess.ErrNotFound {
					return &createEntityProfileOutput{
						ClientMutationID: in.ClientMutationID,
						Success:          false,
						ErrorCode:        profileErrorCodeInvalidMediaID,
						ErrorMessage:     "The provided media id is not valid.",
					}, nil
				} else if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
			}

			org, err := updateOrganizationProfile(ctx, ram, "", in.OrganizationID, in.ImageMediaID, in.DisplayName, in.Sections, svc.staticURLPrefix)
			if lerrors.Cause(err) == raccess.ErrNotFound {
				return nil, errors.ErrNotFound(ctx, fmt.Sprintf("Resource for profile creation for %s", in.OrganizationID))
			} else if err != nil {
				return nil, err
			}

			return &createOrganizationProfileOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          true,
				Organization:     org,
			}, nil
		})),
}

// updateEntityProfile
type updateEntityProfileInput struct {
	ClientMutationID string                 `gql:"clientMutationId"`
	ProfileID        string                 `gql:"profileID,nonempty"`
	DisplayName      string                 `gql:"displayName,nonempty"`
	ImageMediaID     string                 `gql:"imageMediaID"`
	Sections         []*profileSectionInput `gql:"sections,nonempty"`
}

var updateEntityProfileInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateEntityProfileInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"profileID":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"displayName":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"imageMediaID":     &graphql.InputObjectFieldConfig{Type: graphql.String},
			"sections":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(profileSectionInputType))},
		},
	},
)

type updateEntityProfileOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	UUID             string         `json:"uuid,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Entity           *models.Entity `json:"entity"`
}

var updateEntityProfileErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "UpdateEntityProfileErrorCode",
	Values: graphql.EnumValueConfigMap{
		profileErrorCodeInvalidMediaID: &graphql.EnumValueConfig{
			Value:       profileErrorCodeInvalidMediaID,
			Description: "The provided media id is not valid.",
		},
	},
})

var updateEntityProfileOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateEntityProfilePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"uuid":             &graphql.Field{Type: graphql.String},
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: updateEntityProfileErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"entity":           &graphql.Field{Type: entityType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateEntityProfileOutput)
			return ok
		},
	},
)

var updateEntityProfileMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateEntityProfileOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateEntityProfileInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
			svc := serviceFromParams(p)
			ram := raccess.ResourceAccess(p)
			ctx := p.Context

			var in updateEntityProfileInput
			if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, errors.InternalError(ctx, err)
			}

			// Check that our media ID is valid
			if in.ImageMediaID != "" {
				if _, err := ram.MediaInfo(ctx, in.ImageMediaID); lerrors.Cause(err) == raccess.ErrNotFound {
					return &updateEntityProfileOutput{
						ClientMutationID: in.ClientMutationID,
						Success:          false,
						ErrorCode:        profileErrorCodeInvalidMediaID,
						ErrorMessage:     "The provided media id is not valid.",
					}, nil
				} else if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
			}

			ent, err := updateEntityProfile(ctx, ram, in.ProfileID, "", in.ImageMediaID, in.DisplayName, in.Sections, svc.staticURLPrefix)
			if lerrors.Cause(err) == raccess.ErrNotFound {
				return nil, errors.ErrNotFound(ctx, fmt.Sprintf("Resource for profile update %s", in.ProfileID))
			} else if err != nil {
				return nil, err
			}

			return &updateEntityProfileOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          true,
				Entity:           ent,
			}, nil
		})),
}

// updateOrganizationProfile
type updateOrganizationProfileInput struct {
	ClientMutationID string                 `gql:"clientMutationId"`
	ProfileID        string                 `gql:"profileID,nonempty"`
	DisplayName      string                 `gql:"displayName,nonempty"`
	ImageMediaID     string                 `gql:"imageMediaID"`
	Sections         []*profileSectionInput `gql:"sections,nonempty"`
}

var updateOrganizationProfileInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateOrganizationProfileInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"profileID":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"displayName":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"imageMediaID":     &graphql.InputObjectFieldConfig{Type: graphql.String},
			"sections":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(profileSectionInputType))},
		},
	},
)

type updateOrganizationProfileOutput struct {
	ClientMutationID string               `json:"clientMutationId,omitempty"`
	UUID             string               `json:"uuid,omitempty"`
	Success          bool                 `json:"success"`
	ErrorCode        string               `json:"errorCode,omitempty"`
	ErrorMessage     string               `json:"errorMessage,omitempty"`
	Organization     *models.Organization `json:"organization"`
}

var updateOrganizationProfileErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "UpdateOrganizationProfileErrorCode",
	Values: graphql.EnumValueConfigMap{
		profileErrorCodeInvalidMediaID: &graphql.EnumValueConfig{
			Value:       profileErrorCodeInvalidMediaID,
			Description: "The provided media id is not valid.",
		},
	},
})

var updateOrganizationProfileOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateOrganizationProfilePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"uuid":             &graphql.Field{Type: graphql.String},
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: updateOrganizationProfileErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"organization":     &graphql.Field{Type: organizationType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateOrganizationProfileOutput)
			return ok
		},
	},
)

var updateOrganizationProfileMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateOrganizationProfileOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateOrganizationProfileInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
			svc := serviceFromParams(p)
			ram := raccess.ResourceAccess(p)
			ctx := p.Context

			var in updateOrganizationProfileInput
			if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, errors.InternalError(ctx, err)
			}

			if in.ImageMediaID != "" {
				// Check that our media ID is valid
				if _, err := ram.MediaInfo(ctx, in.ImageMediaID); lerrors.Cause(err) == raccess.ErrNotFound {
					return &updateOrganizationProfileOutput{
						ClientMutationID: in.ClientMutationID,
						Success:          false,
						ErrorCode:        profileErrorCodeInvalidMediaID,
						ErrorMessage:     "The provided media id is not valid.",
					}, nil
				} else if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
			}

			org, err := updateOrganizationProfile(ctx, ram, in.ProfileID, "", in.ImageMediaID, in.DisplayName, in.Sections, svc.staticURLPrefix)
			if lerrors.Cause(err) == raccess.ErrNotFound {
				return nil, errors.ErrNotFound(ctx, fmt.Sprintf("Resource for profile update %s", in.ProfileID))
			} else if err != nil {
				return nil, err
			}

			return &updateOrganizationProfileOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          true,
				Organization:     org,
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

	profile, err := ram.UpdateProfile(ctx, &directory.UpdateProfileRequest{
		ProfileID:    profileID,
		ImageMediaID: imageMediaID,
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

func updateOrganizationProfile(ctx context.Context, ram raccess.ResourceAccessor, profileID, organizationID, imageMediaID, customDisplayName string, psis []*profileSectionInput, staticURLPrefix string) (*models.Organization, error) {
	sections := make([]*directory.ProfileSection, len(psis))
	for i, s := range psis {
		sections[i] = &directory.ProfileSection{
			Title: s.Title,
			Body:  s.Body,
		}
	}

	profile, err := ram.UpdateProfile(ctx, &directory.UpdateProfileRequest{
		ProfileID:    profileID,
		ImageMediaID: imageMediaID,
		Profile: &directory.Profile{
			EntityID:    organizationID,
			DisplayName: customDisplayName,
			Sections:    sections,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return transformOrganizationToResponse(ctx, staticURLPrefix, profile.Entity, nil, devicectx.SpruceHeaders(ctx), gqlctx.Account(ctx))
}
