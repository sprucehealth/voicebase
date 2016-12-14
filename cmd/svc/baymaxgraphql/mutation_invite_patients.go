package main

import (
	"context"
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/textutil"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

type invitePatientsOutput struct {
	ClientMutationID string           `json:"clientMutationId,omitempty"`
	Success          bool             `json:"success"`
	ErrorCode        string           `json:"errorCode,omitempty"`
	ErrorMessage     string           `json:"errorMessage,omitempty"`
	PatientThreads   []*models.Thread `json:"patientThreads"`
}

var invitePatientsInfoType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "InvitePatientsInfo",
	Fields: graphql.InputObjectConfigFieldMap{
		"firstName":   &graphql.InputObjectFieldConfig{Type: graphql.String},
		"lastName":    &graphql.InputObjectFieldConfig{Type: graphql.String},
		"email":       &graphql.InputObjectFieldConfig{Type: graphql.String},
		"phoneNumber": &graphql.InputObjectFieldConfig{Type: graphql.String},
	},
})

var invitePatientsInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "InvitePatientsInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"patients":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(invitePatientsInfoType)))},
	},
})

const (
	invitePatientsErrorCodeInvalidFirstName    = "INVALID_FIRST_NAME"
	invitePatientsErrorCodeInvalidLastName     = "INVALID_LAST_NAME"
	invitePatientsErrorCodeInvalidEmail        = "INVALID_EMAIL"
	invitePatientsErrorCodeInvalidPhoneNumber  = "INVALID_PHONE_NUMBER"
	invitePatientsErrorCodeMissingPhoneNumber  = "MISSING_PHONE_NUMBER"
	invitePatientsErrorCodeMissingEmail        = "MISSING_EMAIL"
	invitePatientsErrorCodeMissingEmailOrPhone = "MISSING_EMAIL_OR_PHONE_NUMBER"
)

var invitePatientsErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "InvitePatientsErrorCode",
	Values: graphql.EnumValueConfigMap{
		invitePatientsErrorCodeInvalidFirstName: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeInvalidFirstName,
			Description: "The provided first name is invalid",
		},
		invitePatientsErrorCodeInvalidLastName: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeInvalidLastName,
			Description: "The provided last name is invalid",
		},
		invitePatientsErrorCodeInvalidEmail: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeInvalidEmail,
			Description: "The provided email address is invalid",
		},
		invitePatientsErrorCodeInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeInvalidPhoneNumber,
			Description: "The provided phone number is invalid",
		},
		invitePatientsErrorCodeMissingPhoneNumber: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeMissingPhoneNumber,
			Description: "Phone number is required to create invite",
		},
		invitePatientsErrorCodeMissingEmail: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeMissingEmail,
			Description: "Email is required to create invite",
		},
		invitePatientsErrorCodeMissingEmailOrPhone: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeMissingEmailOrPhone,
			Description: "Either email or phone number is required to create invite",
		},
	},
})

var invitePatientsOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "InvitePatientsPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: invitePatientsErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"patientThreads":   &graphql.Field{Type: graphql.NewList(threadType)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*invitePatientsOutput)
		return ok
	},
})

var invitePatientsMutation = &graphql.Field{
	Description: "invitePatients invites one or more people to an organization",
	Type:        graphql.NewNonNull(invitePatientsOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(invitePatientsInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(
			func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				acc := gqlctx.Account(ctx)

				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				orgID := input["organizationID"].(string)
				patientsInput := input["patients"].([]interface{})

				inviterEnt, err := entityInOrgForAccountID(ctx, ram, orgID, acc)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				if inviterEnt == nil {
					return nil, errors.New("Not a member of the organization")
				}

				settingsRes, err := svc.settings.GetValues(ctx, &settings.GetValuesRequest{
					NodeID: orgID,
					Keys: []*settings.ConfigKey{
						{
							Key: invite.ConfigKeyTwoFactorVerificationForSecureConversation,
						},
					},
				})
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				requirePhoneAndEmailForSecureConversationCreation := settingsRes.Values[0].GetBoolean()

				// Do all our validation in 1 pass
				for _, p := range patientsInput {
					m := p.(map[string]interface{})
					email, _ := m["email"].(string)
					phoneNumber, _ := m["phoneNumber"].(string)

					var firstName string
					iFirstName, ok := m["firstName"]
					if ok {
						firstName = iFirstName.(string)
					}
					var lastName string
					iLastName, ok := m["lastName"]
					if ok {
						lastName = iLastName.(string)
					}

					if email != "" && !validate.Email(email) {
						return &invitePatientsOutput{
							ClientMutationID: mutationID,
							Success:          false,
							ErrorCode:        invitePatientsErrorCodeInvalidEmail,
							ErrorMessage:     fmt.Sprintf("The email address '%s' not valid.", email),
						}, nil
					}

					if phoneNumber != "" {
						var err error
						_, err = phone.Format(phoneNumber, phone.E164)
						if err != nil {
							return &invitePatientsOutput{
								ClientMutationID: mutationID,
								Success:          false,
								ErrorCode:        invitePatientsErrorCodeInvalidPhoneNumber,
								ErrorMessage:     fmt.Sprintf("The phone number '%s' not valid.", phoneNumber),
							}, nil
						}
					}

					if requirePhoneAndEmailForSecureConversationCreation.Value {
						if phoneNumber == "" {
							return &invitePatientsOutput{
								ClientMutationID: mutationID,
								Success:          false,
								ErrorCode:        invitePatientsErrorCodeMissingPhoneNumber,
								ErrorMessage:     fmt.Sprintf("Phone number is required to create invite"),
							}, nil
						}
						if email == "" {
							return &invitePatientsOutput{
								ClientMutationID: mutationID,
								Success:          false,
								ErrorCode:        invitePatientsErrorCodeMissingEmail,
								ErrorMessage:     fmt.Sprintf("Email is required to create invite"),
							}, nil
						}
					} else {
						if phoneNumber == "" && email == "" {
							return &invitePatientsOutput{
								ClientMutationID: mutationID,
								Success:          false,
								ErrorCode:        invitePatientsErrorCodeMissingEmailOrPhone,
								ErrorMessage:     fmt.Sprintf("Email or phone number is required to create invite"),
							}, nil
						}
					}

					if firstName != "" && !textutil.IsValidPlane0Unicode(firstName) {
						return &invitePatientsOutput{
							ClientMutationID: mutationID,
							Success:          false,
							ErrorCode:        invitePatientsErrorCodeInvalidFirstName,
							ErrorMessage:     "Please enter a valid first name.",
						}, nil
					}
					if lastName != "" && !textutil.IsValidPlane0Unicode(lastName) {
						return &invitePatientsOutput{
							ClientMutationID: mutationID,
							Success:          false,
							ErrorCode:        invitePatientsErrorCodeInvalidLastName,
							ErrorMessage:     "Please enter a valid last name.",
						}, nil
					}
				}

				threads := make([]*threading.Thread, len(patientsInput))

				// Next do any writes to prevent partial failure due to validation
				patients := make([]*invite.Patient, 0, len(patientsInput))
				for i, p := range patientsInput {
					m := p.(map[string]interface{})
					var firstName string
					iFirstName, ok := m["firstName"]
					if ok {
						firstName = iFirstName.(string)
					}
					var lastName string
					iLastName, ok := m["lastName"]
					if ok {
						lastName = iLastName.(string)
					}
					pat := &invite.Patient{
						FirstName: firstName,
					}

					// Create a parked entity for the account
					email, _ := m["email"].(string)
					phoneNumber, _ := m["phoneNumber"].(string)
					// Can only ignore this err because we checked it above
					fpn, _ := phone.Format(phoneNumber, phone.E164)
					pat.PhoneNumber = fpn
					pat.Email = email
					patientEntity, err := ram.CreateEntity(ctx, &directory.CreateEntityRequest{
						Type: directory.EntityType_PATIENT,
						InitialMembershipEntityID: orgID,
						Contacts: []*directory.Contact{
							{
								ContactType: directory.ContactType_EMAIL,
								Value:       email,
							},
							{
								ContactType: directory.ContactType_PHONE,
								Value:       pat.PhoneNumber,
							},
						},
						EntityInfo: &directory.EntityInfo{
							FirstName: pat.FirstName,
							LastName:  lastName,
						},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					pat.ParkedEntityID = patientEntity.ID

					// Create a thread with the parked patient in the org
					thread, err := ram.CreateEmptyThread(ctx, &threading.CreateEmptyThreadRequest{
						OrganizationID:  orgID,
						PrimaryEntityID: patientEntity.ID,
						MemberEntityIDs: []string{orgID, patientEntity.ID},
						Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
						Summary:         patientEntity.Info.DisplayName,
						SystemTitle:     patientEntity.Info.DisplayName,
						Origin:          threading.THREAD_ORIGIN_PATIENT_INVITE,
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					threads[i] = thread
					golog.Debugf("Created empty thread %s for parked entity %s", thread.ID, patientEntity.ID)
					patients = append(patients, pat)
				}

				if _, err := svc.invite.InvitePatients(ctx, &invite.InvitePatientsRequest{
					OrganizationEntityID: orgID,
					InviterEntityID:      inviterEnt.ID,
					Patients:             patients,
				}); err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				analytics.SegmentTrack(&segment.Track{
					Event:  "invited-patient",
					UserId: acc.ID,
				})

				patientThreads := make([]*models.Thread, len(threads))
				for i, thread := range threads {
					th, err := transformThreadToResponse(ctx, ram, thread, acc)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					patientThreads[i] = th
				}
				return &invitePatientsOutput{
					ClientMutationID: mutationID,
					Success:          true,
					PatientThreads:   patientThreads,
				}, nil
			})),
}

func contactForParkedEntity(ctx context.Context, ram raccess.ResourceAccessor, parkedEntityID string, contactType directory.ContactType) (string, error) {
	var entityContact string
	// Since we don't store PHI for patients in the invites, get the email to verify from the parked entity contacts
	// Make this as an unauthorized call since we have no context around the caller other than token
	entities, err := ram.Entities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: parkedEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
			},
			Depth: 0,
		},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}, raccess.EntityQueryOptionUnathorized)
	if err != nil {
		return "", fmt.Errorf("Encountered an error while looking up parked entity %q to get %s: %s", parkedEntityID, contactType.String(), err)
	} else if len(entities) > 1 {
		return "", fmt.Errorf("Expected 1 entity to be returned for %s but got back %d", parkedEntityID, len(entities))
	}
	parkedEntity := entities[0]

	for _, c := range parkedEntity.Contacts {
		if c.ContactType == contactType {
			if entityContact != "" {
				golog.Errorf("Parked entity %s had multiple associated %s. Only using first.", parkedEntityID, contactType.String())
				continue
			}
			entityContact = c.Value
		}
	}
	if entityContact == "" {
		return "", fmt.Errorf("Unable to find contact %s for parked entity %q", contactType.String(), parkedEntityID)
	}
	return entityContact, nil
}
