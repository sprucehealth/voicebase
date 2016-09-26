package main

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
)

var networkTypeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "NetworkType",
	Values: graphql.EnumValueConfigMap{
		models.NetworkTypeUnknown: &graphql.EnumValueConfig{
			Value:       models.NetworkTypeUnknown,
			Description: "Unknown network connection",
		},
		models.NetworkTypeCellular: &graphql.EnumValueConfig{
			Value:       models.NetworkTypeCellular,
			Description: "Cellular network connection",
		},
		models.NetworkTypeWiFi: &graphql.EnumValueConfig{
			Value:       models.NetworkTypeWiFi,
			Description: "Wi-Fi network connection",
		},
	},
})

var callRoleEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CallRole",
	Values: graphql.EnumValueConfigMap{
		models.CallRoleCaller: &graphql.EnumValueConfig{
			Value:       models.CallRoleCaller,
			Description: "The person is the caller",
		},
		models.CallRoleRecipient: &graphql.EnumValueConfig{
			Value:       models.CallRoleRecipient,
			Description: "The person is the callee (recipient)",
		},
	},
})

var callStateEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CallState",
	Values: graphql.EnumValueConfigMap{
		models.CallStatePending: &graphql.EnumValueConfig{
			Value:       models.CallStatePending,
			Description: "Call is currently pending",
		},
		models.CallStateAccepted: &graphql.EnumValueConfig{
			Value:       models.CallStateAccepted,
			Description: "Recipient has indicated that they want to accept the call, and has connected to twilio",
		},
		models.CallStateDeclined: &graphql.EnumValueConfig{
			Value:       models.CallStateDeclined,
			Description: "Recipient has declined the call",
		},
		models.CallStateConnected: &graphql.EnumValueConfig{
			Value:       models.CallStateConnected,
			Description: "Party has confirmed that they successfully connected to the Twilio call",
		},
		models.CallStateFailed: &graphql.EnumValueConfig{
			Value:       models.CallStateFailed,
			Description: "Party failed to connect to the Twilio call",
		},
		models.CallStateCompleted: &graphql.EnumValueConfig{
			Value:       models.CallStateCompleted,
			Description: "Call ended successfully",
		},
	},
})

var callChannelTypeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CallChannelType",
	Values: graphql.EnumValueConfigMap{
		models.CallChannelTypePhone: &graphql.EnumValueConfig{
			Value:       models.CallChannelTypePhone,
			Description: "Traditional phone call",
		},
		models.CallChannelTypeVOIP: &graphql.EnumValueConfig{
			Value:       models.CallChannelTypeVOIP,
			Description: "Voice over IP via Twilio",
		},
		models.CallChannelTypeVideo: &graphql.EnumValueConfig{
			Value:       models.CallChannelTypeVideo,
			Description: "Video call via Twilio",
		},
	},
})

var callableIdentityType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CallableIdentity",
	Fields: graphql.Fields{
		"name":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"endpoints": &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(callEndpointType)))},
		"entity":    &graphql.Field{Type: entityType},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*models.CallableIdentity)
		return ok
	},
})

var callEndpointType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CallEndpoint",
	Fields: graphql.Fields{
		"channel":                 &graphql.Field{Type: graphql.NewNonNull(callChannelTypeEnum)},
		"displayValue":            &graphql.Field{Type: graphql.String},
		"valueOrID":               &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"lanConnectivityRequired": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"label":                   &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*models.CallEndpoint)
		return ok
	},
})

var callType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Call",
	Interfaces: []*graphql.Interface{
		nodeInterfaceType,
	},
	Fields: graphql.Fields{
		"id":                      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"accessToken":             &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"role":                    &graphql.Field{Type: graphql.NewNonNull(callRoleEnum)},
		"caller":                  &graphql.Field{Type: graphql.NewNonNull(callParticipantType)},
		"recipients":              &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(callParticipantType))},
		"allowVideo":              &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"videoEnabledByDefault":   &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"lanConnectivityRequired": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*models.Call)
		return ok
	},
})

var callParticipantType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CallParticipant",
	Fields: graphql.Fields{
		"entity": &graphql.Field{
			Type: graphql.NewNonNull(entityType),
			Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
				ctx := p.Context
				ram := raccess.ResourceAccess(p)
				svc := serviceFromParams(p)
				acc := gqlctx.Account(p.Context)
				par := p.Source.(*models.CallParticipant)
				ent, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: par.EntityID,
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
					},
					Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
				})
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				return transformEntityToResponse(ctx, svc.staticURLPrefix, ent, devicectx.SpruceHeaders(ctx), acc)
			}),
		},
		"state":          &graphql.Field{Type: graphql.NewNonNull(callStateEnum)},
		"twilioIdentity": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"networkType":    &graphql.Field{Type: graphql.NewNonNull(networkTypeEnum)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*models.CallParticipant)
		return ok
	},
})

func lookupCall(ctx context.Context, ram raccess.ResourceAccessor, id string) (*models.Call, error) {
	call, err := ram.IPCall(ctx, id)
	if err != nil {
		return nil, err
	}
	acc := gqlctx.Account(ctx)
	return transformCallToResponse(call, acc.ID)
}

func callableEndpointsForEntity(ctx context.Context, ent *directory.Entity) ([]*models.CallEndpoint, error) {
	var endpoints []*models.CallEndpoint
	if ent.Type == directory.EntityType_PATIENT || (ent.Type == directory.EntityType_INTERNAL && !environment.IsProd() && !environment.IsTest()) {
		if gqlctx.FeatureEnabled(ctx, gqlctx.VideoCalling) {
			endpoints = append(endpoints, &models.CallEndpoint{
				Channel:                 models.CallChannelTypeVideo,
				ValueOrID:               ent.ID,
				LANConnectivityRequired: false,
			})
		}
	}
	for _, c := range ent.Contacts {
		if c.ContactType == directory.ContactType_PHONE {
			display, err := phone.Format(c.Value, phone.Pretty)
			if err != nil {
				golog.Errorf("Failed to format phone number of contact %s for entity %s: %s", c.ID, ent.ID, err)
				continue
			}
			endpoints = append(endpoints, &models.CallEndpoint{
				Channel:      models.CallChannelTypePhone,
				DisplayValue: display,
				ValueOrID:    c.Value,
				Label:        c.Label,
			})
		}
	}
	return endpoints, nil
}

func transformCallToResponse(call *excomms.IPCall, accountID string) (*models.Call, error) {
	if len(call.Participants) != 2 {
		return nil, fmt.Errorf("Expected 2 participants for call %s, got %d", call.ID, len(call.Participants))
	}
	c := &models.Call{
		ID:                      call.ID,
		AccessToken:             call.Token,
		AllowVideo:              true,
		VideoEnabledByDefault:   true,
		Recipients:              make([]*models.CallParticipant, 0, len(call.Participants)-1),
		LANConnectivityRequired: false,
	}
	for _, p := range call.Participants {
		par := &models.CallParticipant{
			EntityID:       p.EntityID,
			TwilioIdentity: p.Identity,
			State:          p.State.String(),
			NetworkType:    p.NetworkType.String(),
		}
		switch p.Role {
		case excomms.IPCallParticipantRole_CALLER:
			c.Caller = par
		case excomms.IPCallParticipantRole_RECIPIENT:
			c.Recipients = append(c.Recipients, par)
		default:
			return nil, fmt.Errorf("Unknown ipcall participant role '%s'", p.Role)
		}
		if p.AccountID == accountID {
			c.Role = p.Role.String()
		}
	}
	return c, nil
}

func parseNetworkTypeInput(nt string) (excomms.NetworkType, error) {
	switch nt {
	case models.NetworkTypeUnknown:
		return excomms.NetworkType_UNKNOWN, nil
	case models.NetworkTypeCellular:
		return excomms.NetworkType_CELLULAR, nil
	case models.NetworkTypeWiFi:
		return excomms.NetworkType_WIFI, nil
	}
	return excomms.NetworkType_UNKNOWN, errors.Errorf("unknown network type %s", nt)
}

func parseCallStateInput(cs string) (excomms.IPCallState, error) {
	switch cs {
	case models.CallStatePending:
		return excomms.IPCallState_PENDING, nil
	case models.CallStateAccepted:
		return excomms.IPCallState_ACCEPTED, nil
	case models.CallStateDeclined:
		return excomms.IPCallState_DECLINED, nil
	case models.CallStateConnected:
		return excomms.IPCallState_CONNECTED, nil
	case models.CallStateFailed:
		return excomms.IPCallState_FAILED, nil
	case models.CallStateCompleted:
		return excomms.IPCallState_COMPLETED, nil
	}
	return excomms.IPCallState_INVALID_IPCALL_STATE, errors.Errorf("unknown call state %s", cs)
}
