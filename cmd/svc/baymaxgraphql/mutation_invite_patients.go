package main

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
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
		"firstName":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"lastName":    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"email":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"phoneNumber": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
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
	invitePatientsErrorCodeInvalidFirstName   = "INVALID_FIRST_NAME"
	invitePatientsErrorCodeInvalidEmail       = "INVALID_EMAIL"
	invitePatientsErrorCodeInvalidPhoneNumber = "INVALID_PHONE_NUMBER"
)

var invitePatientsErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "InvitePatientsErrorCode",
	Values: graphql.EnumValueConfigMap{
		invitePatientsErrorCodeInvalidFirstName: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeInvalidFirstName,
			Description: "The provided first name is invalid",
		},
		invitePatientsErrorCodeInvalidEmail: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeInvalidEmail,
			Description: "The provided email address is invalid",
		},
		invitePatientsErrorCodeInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       invitePatientsErrorCodeInvalidPhoneNumber,
			Description: "The provided phone number is invalid",
		},
	},
})

var invitePatientsOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "InvitePatientsPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
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
				inviterEnt, err := ram.EntityForAccountID(ctx, orgID, acc.ID)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				if inviterEnt == nil {
					return nil, errors.New("Not a member of the organization")
				}

				// Do all our validation in 1 pass
				for _, p := range patientsInput {
					m := p.(map[string]interface{})
					email := m["email"].(string)
					phoneNumber := m["phoneNumber"].(string)
					firstName := m["firstName"].(string)
					if !validate.Email(email) {
						return &invitePatientsOutput{
							ClientMutationID: mutationID,
							Success:          false,
							ErrorCode:        invitePatientsErrorCodeInvalidEmail,
							ErrorMessage:     fmt.Sprintf("The email address '%s' not valid.", email),
						}, nil
					}
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
					if firstName == "" || !isValidPlane0Unicode(firstName) {
						return &createPatientAccountOutput{
							ClientMutationID: mutationID,
							Success:          false,
							ErrorCode:        invitePatientsErrorCodeInvalidFirstName,
							ErrorMessage:     "Please enter a valid first name.",
						}, nil
					}
				}

				threads := make([]*threading.Thread, len(patientsInput))

				// Next do any writes to prevent partial failure due to validation
				patients := make([]*invite.Patient, 0, len(patientsInput))
				for i, p := range patientsInput {
					m := p.(map[string]interface{})
					pat := &invite.Patient{
						FirstName: m["firstName"].(string),
					}

					// Create a parked entity for the account
					lastName := m["lastName"].(string)
					email := m["email"].(string)
					// Can only ignore this err because we checked it above
					fpn, _ := phone.Format(m["phoneNumber"].(string), phone.E164)
					pat.PhoneNumber = fpn
					patientEntity, err := ram.CreateEntity(ctx, &directory.CreateEntityRequest{
						Type: directory.EntityType_PATIENT,
						InitialMembershipEntityID: orgID,
						Contacts: []*directory.Contact{
							&directory.Contact{
								ContactType: directory.ContactType_EMAIL,
								Value:       email,
							},
							&directory.Contact{
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
						MemberEntityIDs: []string{inviterEnt.ID, patientEntity.ID},
						Type:            threading.ThreadType_SECURE_EXTERNAL,
						Summary:         patientEntity.Info.DisplayName,
						SystemTitle:     patientEntity.Info.DisplayName,
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

				svc.segmentio.Track(&analytics.Track{
					Event:  "invited-patient",
					UserId: acc.ID,
				})

				patientThreads := make([]*models.Thread, len(threads))
				for i, thread := range threads {
					th, err := transformThreadToResponse(thread, acc)
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
	parkedEntity, err := ram.Entity(ctx, parkedEntityID, []directory.EntityInformation{directory.EntityInformation_CONTACTS}, 0)
	if err != nil {
		return "", fmt.Errorf("Encountered an error while looking up parked entity %q to get %s: %s", parkedEntityID, contactType.String(), err)
	}
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
