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
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

var sendExistingPatientInviteInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "SendExistingPatientInviteInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"phoneContactID":   &graphql.InputObjectFieldConfig{Type: graphql.ID},
		"emailContactID":   &graphql.InputObjectFieldConfig{Type: graphql.ID},
	},
})

type sendExistingPatientInviteInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	EntityID         string `gql:"entityID"`
	PhoneContactID   string `gql:"phoneContactID"`
	EmailContactID   string `gql:"emailContactID"`
	OrganizationID   string `gql:"organizationID"`
}

const (
	sendPatientInviteErrorCodeInvalidPhoneContactID = "INVALID_PHONE_CONTACT_ID"
	sendPatientInviteErrorCodeInvalidEmailContactID = "INVALID_EMAIL_CONTACT_ID"
	sendPaitentInviteErrorCodePhoneNumberNotFound   = "PHONE_NUMBER_NOT_FOUND"
	sendPaitentInviteErrorCodeEmailNotFound         = "EMAIL_NOT_FOUND"
)

var sendExistingPatientInviteErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "SendExistingPatientInviteErrorCode",
	Values: graphql.EnumValueConfigMap{
		sendPatientInviteErrorCodeInvalidPhoneContactID: &graphql.EnumValueConfig{
			Value:       sendPatientInviteErrorCodeInvalidPhoneContactID,
			Description: "The provided phone contact ID is invalid.",
		},
		sendPatientInviteErrorCodeInvalidEmailContactID: &graphql.EnumValueConfig{
			Value:       sendPatientInviteErrorCodeInvalidEmailContactID,
			Description: "The provided email contact ID is invalid.",
		},
		sendPaitentInviteErrorCodePhoneNumberNotFound: &graphql.EnumValueConfig{
			Value:       sendPaitentInviteErrorCodePhoneNumberNotFound,
			Description: "Phone number not found",
		},
		sendPaitentInviteErrorCodeEmailNotFound: &graphql.EnumValueConfig{
			Value:       sendPaitentInviteErrorCodeEmailNotFound,
			Description: "Email not found",
		},
	},
})

type sendExistingPatientInviteOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var sendExistingPatientInviteOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "SendExistingPatientInvitePayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: sendExistingPatientInviteErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*sendExistingPatientInviteOutput)
		return ok
	},
})

var sendExistingPatientInviteMutation = &graphql.Field{
	Description: "sendExistingPatientInviteMutation enables a provider to (re)send a patient invite via SMS for secure conversations",
	Type:        graphql.NewNonNull(sendExistingPatientInviteOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(sendExistingPatientInviteInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(
			func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				acc := gqlctx.Account(ctx)

				input := p.Args["input"].(map[string]interface{})
				var in sendExistingPatientInviteInput
				if err := gqldecode.Decode(input, &in); err != nil {
					switch err := err.(type) {
					case gqldecode.ErrValidationFailed:
						return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
					}
					return nil, errors.InternalError(ctx, err)
				}

				patientEntity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: in.EntityID,
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{
							directory.EntityInformation_CONTACTS,
							directory.EntityInformation_MEMBERSHIPS,
						},
					},
				})
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				} else if patientEntity.AccountID != "" {
					// nothing to do here since the account has already been created
					return &sendExistingPatientInviteOutput{
						Success:          true,
						ClientMutationID: in.ClientMutationID,
					}, nil
				}

				entity, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
						ExternalID: acc.ID,
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
					},
					Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
				}, in.OrganizationID)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				var phoneNumber, email string
				for _, contact := range patientEntity.Contacts {
					if contact.ID == in.PhoneContactID {
						if contact.ContactType != directory.ContactType_PHONE {
							return &sendExistingPatientInviteOutput{
								ClientMutationID: in.ClientMutationID,
								Success:          false,
								ErrorCode:        sendPatientInviteErrorCodeInvalidPhoneContactID,
								ErrorMessage:     fmt.Sprintf("Specified contact ID is not a phone number but a %s", contact.ContactType),
							}, nil
						}
						phoneNumber = contact.Value
					} else if contact.ID == in.EmailContactID {
						if contact.ContactType != directory.ContactType_EMAIL {
							return &sendExistingPatientInviteOutput{
								ClientMutationID: in.ClientMutationID,
								Success:          false,
								ErrorCode:        sendPatientInviteErrorCodeInvalidEmailContactID,
								ErrorMessage:     fmt.Sprintf("Specified contact ID is not an email address but a %s", contact.ContactType),
							}, nil
						}
						email = contact.Value
					}

					// pick first available email and phone if particular one not specified
					if phoneNumber == "" && in.PhoneContactID == "" && contact.ContactType == directory.ContactType_PHONE {
						phoneNumber = contact.Value
					}
					if email == "" && in.EmailContactID == "" && contact.ContactType == directory.ContactType_EMAIL {
						email = contact.Value
					}
				}

				if phoneNumber == "" {
					return &sendExistingPatientInviteOutput{
						Success:      false,
						ErrorCode:    sendPaitentInviteErrorCodePhoneNumberNotFound,
						ErrorMessage: "No phone number found for patient",
					}, nil
				}

				if email == "" {
					return &sendExistingPatientInviteOutput{
						Success:      false,
						ErrorCode:    sendPaitentInviteErrorCodeEmailNotFound,
						ErrorMessage: "No phone number found for patient",
					}, nil
				}

				if _, err := svc.invite.InvitePatients(ctx, &invite.InvitePatientsRequest{
					OrganizationEntityID: in.OrganizationID,
					InviterEntityID:      entity.ID,
					Patients: []*invite.Patient{&invite.Patient{
						FirstName:      patientEntity.Info.FirstName,
						PhoneNumber:    phoneNumber,
						Email:          email,
						ParkedEntityID: patientEntity.ID,
					}},
				}); err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				analytics.SegmentTrack(&segment.Track{
					Event:  "send-patient-invite",
					UserId: acc.ID,
				})

				return &sendExistingPatientInviteOutput{
					Success:          true,
					ClientMutationID: in.ClientMutationID,
				}, nil
			},
		)),
}
