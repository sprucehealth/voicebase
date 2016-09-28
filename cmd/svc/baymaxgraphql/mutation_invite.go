package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/invite/clientdata"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// associateAttribution

type associateAttributionOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var associateAttributionValueType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "AssociateAttributionValue",
	Fields: graphql.InputObjectConfigFieldMap{
		"key":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"value": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

var associateAttributionInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AssociateAttributionInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"values":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(associateAttributionValueType)))},
		},
	},
)

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var associateAttributionErrorCodeEnum = graphql.String

var associateAttributionOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AssociateAttributionPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: associateAttributionErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*associateAttributionOutput)
			return ok
		},
	},
)

var associateAttributionMutation = &graphql.Field{
	Description: "associateAttribution attaches attribution information to the device ID of the requester",
	Type:        graphql.NewNonNull(associateAttributionOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(associateAttributionInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		sh := devicectx.SpruceHeaders(ctx)
		if sh.DeviceID == "" {
			return nil, errors.New("missing device ID")
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		valuesInput := input["values"].([]interface{})
		values := make([]*invite.AttributionValue, len(valuesInput))
		for i, v := range valuesInput {
			m := v.(map[string]interface{})
			value, _ := m["value"].(string)
			if value != "" {
				values[i] = &invite.AttributionValue{
					Key:   m["key"].(string),
					Value: value,
				}
			}
		}
		_, err := svc.invite.SetAttributionData(ctx, &invite.SetAttributionDataRequest{
			DeviceID: sh.DeviceID,
			Values:   values,
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &associateAttributionOutput{ClientMutationID: mutationID, Success: true}, nil
	},
}

// associateInvite

var inviteValueType = graphql.NewObject(graphql.ObjectConfig{
	Name: "InviteValue",
	Fields: graphql.Fields{
		"key":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"value": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

type inviteValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type associateInviteOutput struct {
	ClientMutationID string        `json:"clientMutationId,omitempty"`
	Success          bool          `json:"success"`
	ErrorCode        string        `json:"errorCode,omitempty"`
	ErrorMessage     string        `json:"errorMessage,omitempty"`
	InviteType       string        `json:"inviteType"`
	Values           []inviteValue `json:"values,omitempty"`
}

var associateInviteInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AssociateInviteInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

const associateInviteErrorCodeInvalidInvite = "INVALID_INVITE"

var associateInviteErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "AssociateInviteErrorCode",
	Values: graphql.EnumValueConfigMap{
		associateInviteErrorCodeInvalidInvite: &graphql.EnumValueConfig{
			Value:       associateInviteErrorCodeInvalidInvite,
			Description: "The provided token doesn't match a valid invite",
		},
	},
})

var associateInviteOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AssociateInvitePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: associateInviteErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"inviteType":       &graphql.Field{Type: inviteTypeEnum},
			"values": &graphql.Field{
				Type:        graphql.NewList(graphql.NewNonNull(inviteValueType)),
				Description: "Values is the set of data attached to the invite which matters the attribution data from Branch",
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*associateInviteOutput)
			return ok
		},
	},
)

const (
	inviteTypeUnknown          = "UNKNOWN"
	inviteTypePatient          = "PATIENT"
	inviteTypeColleague        = "COLLEAGUE"
	inviteTypeOrganizationCode = "ORGANIZATION_CODE"
)

var inviteTypeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "InviteType",
	Values: graphql.EnumValueConfigMap{
		inviteTypeUnknown: &graphql.EnumValueConfig{
			Value:       inviteTypeUnknown,
			Description: "Indicates that the provided invite code was mapped to an unknown type",
		},
		inviteTypePatient: &graphql.EnumValueConfig{
			Value:       inviteTypePatient,
			Description: "Indicates that the provided invite code was for a patient invite",
		},
		inviteTypeColleague: &graphql.EnumValueConfig{
			Value:       inviteTypeColleague,
			Description: "Indicates that the provided invite code was for a provider invite",
		},
		inviteTypeOrganizationCode: &graphql.EnumValueConfig{
			Value:       inviteTypeOrganizationCode,
			Description: "Indicates that the provided invite code was for an organization code",
		},
	},
})

var associateInviteMutation = &graphql.Field{
	Description: "associateInvite looks up an invite by token, attaches the attribution data to the device ID, and returns the attribution data",
	Type:        graphql.NewNonNull(associateInviteOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(associateInviteInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		ram := raccess.ResourceAccess(p)
		sh := devicectx.SpruceHeaders(ctx)
		if sh.DeviceID == "" {
			return nil, errors.New("missing device ID")
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token := input["token"].(string)
		res, err := svc.invite.LookupInvite(ctx, &invite.LookupInviteRequest{
			LookupKeyType: invite.LookupInviteRequest_TOKEN,
			LookupKeyOneof: &invite.LookupInviteRequest_Token{
				Token: token,
			},
		})
		if grpc.Code(err) == codes.NotFound {
			return &associateInviteOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        associateInviteErrorCodeInvalidInvite,
				ErrorMessage:     "Sorry, the invite code you entered is not valid. Please re-enter the code or contact your healthcare provider.",
			}, nil
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		var orgID string
		var firstName string
		switch res.Type {
		case invite.LookupInviteResponse_PATIENT:
			firstName = res.GetPatient().Patient.FirstName
			orgID = res.GetPatient().OrganizationEntityID
		case invite.LookupInviteResponse_COLLEAGUE:
			firstName = res.GetColleague().Colleague.FirstName
			orgID = res.GetColleague().OrganizationEntityID
		case invite.LookupInviteResponse_ORGANIZATION_CODE:
			orgID = res.GetOrganization().OrganizationEntityID
		default:
			return nil, errors.InternalError(ctx, fmt.Errorf("Unknown invite type %s", res.Type))
		}

		org, err := raccess.UnauthorizedEntity(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: orgID,
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("Error while looking up org %q for device association: %s", orgID, err))
		}

		var clientData string
		switch res.Type {
		case invite.LookupInviteResponse_PATIENT, invite.LookupInviteResponse_ORGANIZATION_CODE:
			var meta *media.MediaInfo
			if org.ImageMediaID != "" {

				meta, err = ram.UpdateMedia(ctx, &media.UpdateMediaRequest{
					MediaID: org.ImageMediaID,
					Public:  true,
				})
				if err != nil {
					return err, errors.InternalError(ctx, err)
				}
			}

			var mimeType string
			if meta != nil {
				mimeType = media.MIMEType(meta.MIME)
			}

			clientData, err = clientdata.PatientInviteClientJSON(org, firstName, svc.mediaAPIDomain, mimeType, res.Type)
			if err != nil {
				golog.Errorf("Error while generating client data for invite to org %s: %s", org.ID, err)
			}
		case invite.LookupInviteResponse_COLLEAGUE:
			inviter, err := raccess.UnauthorizedEntity(ctx, ram, &directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: res.GetColleague().InviterEntityID,
				},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, fmt.Errorf("Error while looking up inviter %s for device association: %s", res.GetColleague().InviterEntityID, err))
			}

			var mimeType string
			if org.ImageMediaID != "" {
				meta, err := ram.MediaInfo(ctx, org.ImageMediaID)
				if err != nil {
					return err, errors.InternalError(ctx, err)
				}
				mimeType = media.MIMEType(meta.MIME)
			}

			clientData, err = clientdata.ColleagueInviteClientJSON(org, inviter, firstName, svc.mediaAPIDomain, mimeType)
			if err != nil {
				golog.Errorf("Error while generating client data for invite to org %s: %s", org.ID, err)
			}
		default:
			return nil, errors.InternalError(ctx, fmt.Errorf("Unknown invite type %s", res.Type))
		}
		// Be backwards compatible with client data and type population
		var foundClientData bool
		var foundType bool
		for _, v := range res.Values {
			switch v.Key {
			case "client_data":
				foundClientData = true
				v.Value = clientData
			case "invite_type":
				foundType = true
				v.Value = res.Type.String()
			}
		}
		if !foundClientData {
			res.Values = append(res.Values, &invite.AttributionValue{Key: "client_data", Value: clientData})
		}
		if !foundType {
			res.Values = append(res.Values, &invite.AttributionValue{Key: "invite_type", Value: res.Type.String()})
		}

		if _, err := svc.invite.SetAttributionData(ctx, &invite.SetAttributionDataRequest{
			DeviceID: sh.DeviceID,
			Values:   res.Values,
		}); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		values := make([]inviteValue, len(res.Values))
		for i, v := range res.Values {
			values[i] = inviteValue{Key: v.Key, Value: v.Value}
		}

		return &associateInviteOutput{
			ClientMutationID: mutationID,
			Success:          true,
			InviteType:       inviteTypeToEnum(res.Type),
			Values:           values,
		}, nil
	},
}

func inviteTypeToEnum(t invite.LookupInviteResponse_Type) string {
	switch t {
	case invite.LookupInviteResponse_PATIENT:
		return inviteTypePatient
	case invite.LookupInviteResponse_COLLEAGUE:
		return inviteTypeColleague
	case invite.LookupInviteResponse_ORGANIZATION_CODE:
		return inviteTypeOrganizationCode
	default:
		golog.Errorf("Unknown invite type %s, returning unknown", t.String())
	}
	return inviteTypeUnknown
}
