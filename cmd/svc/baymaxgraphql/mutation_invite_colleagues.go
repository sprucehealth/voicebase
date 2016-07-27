package main

import (
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

type inviteColleaguesOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

type inviteColleaguesInfoInput struct {
	FirstName   string `gql:"firstName"`
	LastName    string `gql:"lastName"`
	Email       string `gql:"email,nonempty"`
	PhoneNumber string `gql:"phoneNumber,nonempty"`
}

var inviteColleaguesInfoType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "InviteColleaguesInfo",
	Fields: graphql.InputObjectConfigFieldMap{
		// TODO: For now existing clients won't use these fields
		"firstName":   &graphql.InputObjectFieldConfig{Type: graphql.String},
		"lastName":    &graphql.InputObjectFieldConfig{Type: graphql.String},
		"email":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"phoneNumber": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

type inviteColleaguesInput struct {
	ClientMutationID string                       `gql:"clientMutationId"`
	OrganizationID   string                       `gql:"organizationID,nonempty"`
	Colleagues       []*inviteColleaguesInfoInput `gql:"colleagues,nonempty"`
}

var inviteColleaguesInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "InviteColleaguesInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"colleagues":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(inviteColleaguesInfoType)))},
	},
})

const (
	inviteColleaguesErrorCodeInvalidEmail       = "INVALID_EMAIL"
	inviteColleaguesErrorCodeInvalidPhoneNumber = "INVALID_PHONE_NUMBER"
)

var inviteColleaguesErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "InviteColleaguesErrorCode",
	Values: graphql.EnumValueConfigMap{
		inviteColleaguesErrorCodeInvalidEmail: &graphql.EnumValueConfig{
			Value:       inviteColleaguesErrorCodeInvalidEmail,
			Description: "The provided email address is invalid",
		},
		inviteColleaguesErrorCodeInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       inviteColleaguesErrorCodeInvalidPhoneNumber,
			Description: "The provided phone number is invalid",
		},
	},
})

var inviteColleaguesOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "InviteColleaguesPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: inviteColleaguesErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*inviteColleaguesOutput)
		return ok
	},
})

var inviteColleaguesMutation = &graphql.Field{
	Description: "inviteColleagues invites one or more people to an organization",
	Type:        graphql.NewNonNull(inviteColleaguesOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(inviteColleaguesInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(
			func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				acc := gqlctx.Account(ctx)

				var in inviteColleaguesInput
				if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
					switch err := err.(type) {
					case gqldecode.ErrValidationFailed:
						return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
					}
					return nil, errors.InternalError(ctx, err)
				}

				// Independently validate all the inputs before performing network actions
				colleagues := make([]*invite.Colleague, len(in.Colleagues))
				for i, c := range in.Colleagues {
					col := &invite.Colleague{
						Email:       c.Email,
						PhoneNumber: c.PhoneNumber,
						FirstName:   c.FirstName,
					}
					if !validate.Email(col.Email) {
						return &inviteColleaguesOutput{
							ClientMutationID: in.ClientMutationID,
							Success:          false,
							ErrorCode:        inviteColleaguesErrorCodeInvalidEmail,
							ErrorMessage:     fmt.Sprintf("The email address '%s' not valid.", col.Email),
						}, nil
					}
					var err error
					col.PhoneNumber, err = phone.Format(col.PhoneNumber, phone.E164)
					if err != nil {
						return &inviteColleaguesOutput{
							ClientMutationID: in.ClientMutationID,
							Success:          false,
							ErrorCode:        inviteColleaguesErrorCodeInvalidEmail,
							ErrorMessage:     fmt.Sprintf("The phone number '%s' not valid.", col.PhoneNumber),
						}, nil
					}
					colleagues[i] = col
				}

				// Validate that our caller can do this
				ent, err := entityInOrgForAccountID(ctx, ram, in.OrganizationID, acc)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				if ent == nil {
					return nil, errors.New("Not a member of the organization")
				}

				if _, err := svc.invite.InviteColleagues(ctx, &invite.InviteColleaguesRequest{
					OrganizationEntityID: in.OrganizationID,
					InviterEntityID:      ent.ID,
					Colleagues:           colleagues,
				}); err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				for _, c := range colleagues {
					analytics.SegmentTrack(&segment.Track{
						Event:  "invited-colleague",
						UserId: acc.ID,
						Properties: map[string]interface{}{
							"email":        c.Email,
							"phone_number": c.PhoneNumber,
						},
					})
				}

				return &inviteColleaguesOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          true,
				}, nil
			})),
}
