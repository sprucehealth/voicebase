package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	exsettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var savedThreadQueriesField = &graphql.Field{
	Type: graphql.NewList(graphql.NewNonNull(savedThreadQueryType)),
	Args: graphql.FieldConfigArgument{
		"withHidden": &graphql.ArgumentConfig{Type: graphql.Boolean},
	},
	Resolve: apiaccess.Provider(
		func(p graphql.ResolveParams) (interface{}, error) {
			var entityID string
			switch s := p.Source.(type) {
			case *models.Organization:
				if s.Entity == nil || s.Entity.ID == "" {
					return nil, errors.New("no entity for organization")
				}
				entityID = s.Entity.ID
			case *markThreadsAsReadOutput:
				entityID = s.entity.ID
			}

			// withHidden defaults to false if not provided
			withHidden, _ := p.Args["withHidden"].(bool)

			ram := raccess.ResourceAccess(p)
			ctx := p.Context
			sqs, err := ram.SavedQueries(ctx, entityID)
			if err != nil {
				return nil, err
			}
			qs := make([]*models.SavedThreadQuery, 0, len(sqs))
			for _, q := range sqs {
				if q.Type == threading.SAVED_QUERY_TYPE_NORMAL && (withHidden || !q.Hidden) {
					sq, err := transformSavedQueryToResponse(q)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					qs = append(qs, sq)
				}
			}
			return qs, nil
		}),
}

const (
	secureConversationCreationRequirementPhone         = "PHONE_REQUIRED"
	secureConversationCreationRequirementEmail         = "EMAIL_REQUIRED"
	secureConversationCreationRequirementPhoneAndEmail = "PHONE_AND_EMAIL_REQUIRED"
	secureConversationCreationRequirementPhoneOrEmail  = "PHONE_OR_EMAIL_REQUIRED"
)

var secureConversationCreationRequirementEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "SecureConversationCreationRequirement",
	Values: graphql.EnumValueConfigMap{
		secureConversationCreationRequirementEmail: &graphql.EnumValueConfig{
			Value:       secureConversationCreationRequirementEmail,
			Description: "Patient email required for secure conversation creation",
		},
		secureConversationCreationRequirementPhone: &graphql.EnumValueConfig{
			Value:       secureConversationCreationRequirementPhone,
			Description: "Patient phone required for secure conversation creation",
		},
		secureConversationCreationRequirementPhoneAndEmail: &graphql.EnumValueConfig{
			Value:       secureConversationCreationRequirementPhoneAndEmail,
			Description: "Paitent email and phone required for secure conversation creation",
		},
		secureConversationCreationRequirementPhoneOrEmail: &graphql.EnumValueConfig{
			Value:       secureConversationCreationRequirementPhoneOrEmail,
			Description: "Patient email or phone required for secure conversation creation",
		},
	},
})

var organizationType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Organization",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"allowTeamConversations": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					org := p.Source.(*models.Organization)
					if org == nil {
						return false, nil
					}

					booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
						NodeID: org.ID,
						Keys: []*settings.ConfigKey{
							{
								Key: baymaxgraphqlsettings.ConfigKeyTeamConversations,
							},
						},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					return booleanValue.Value, nil
				},
			},
			"allowFilteredTabsInInbox": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					org := p.Source.(*models.Organization)
					if org == nil {
						return false, nil
					}

					booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
						NodeID: org.ID,
						Keys: []*settings.ConfigKey{
							{
								Key: baymaxgraphqlsettings.ConfigKeyFilteredTabsInInbox,
							},
						},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					return booleanValue.Value, nil
				},
			},
			"allowShakeToMarkThreadsAsRead": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					if org.Entity == nil {
						return false, nil
					}

					svc := serviceFromParams(p)
					ctx := p.Context
					acc := gqlctx.Account(ctx)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}

					boolValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
						NodeID: org.ID,
						Keys: []*settings.ConfigKey{
							{
								Key: baymaxgraphqlsettings.ConfigKeyShakeToMarkThreadsAsRead,
							},
						},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					return boolValue.Value, nil
				},
			},
			"allowCreateSecureThread": &graphql.Field{
				Type:    graphql.NewNonNull(graphql.Boolean),
				Resolve: isSecureThreadsEnabled(),
			},
			"entity": &graphql.Field{
				Type:    entityType,
				Resolve: entityWithinOrg(),
			},
			"myEntity": &graphql.Field{
				Type:    entityType,
				Resolve: entityWithinOrg(),
			},
			"contacts": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(contactInfoType)),
				Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					ram := raccess.ResourceAccess(p)
					svc := serviceFromParams(p)
					ctx := p.Context
					acc := gqlctx.Account(ctx)

					if acc.Type == auth.AccountType_PATIENT {
						return nil, nil
					}

					// Get the entity for the account
					ent, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
						Key: &directory.LookupEntitiesRequest_AccountID{
							AccountID: acc.ID,
						},
						RequestedInformation: &directory.RequestedInformation{
							Depth:             0,
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
						},
						Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
						RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
						ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
					}, org.ID)
					if err != nil {
						return nil, errors.Trace(err)
					}

					// Get the default provisioned number for the entity if one is set
					var provisionedPhoneNumber string
					val, err := settings.GetTextValue(ctx, svc.settings, &settings.GetValuesRequest{
						NodeID: ent.ID,
						Keys: []*settings.ConfigKey{
							{
								Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
							},
						},
					})
					if err == nil {
						provisionedPhoneNumber = val.Value
					} else if errors.Cause(err) != settings.ErrValueNotFound {
						return nil, errors.Errorf("unable to get default number setting for entity %s: %s", ent.ID, err)
					}

					// If entity has a default number then order the contacts to have it at the top
					if provisionedPhoneNumber != "" {
						for i, c := range org.Contacts {
							if c.Type == models.ContactTypePhone && c.Provisioned && c.Value == provisionedPhoneNumber {
								if i != 0 { // if not already at the front then swap
									org.Contacts[0], org.Contacts[i] = org.Contacts[i], org.Contacts[0]
								}
								break
							}
						}
					}

					return org.Contacts, nil
				}),
			},
			"entities": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(entityType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					if org.Entity == nil || org.Entity.ID == "" {
						return nil, errors.New("no entity for organization")
					}
					ram := raccess.ResourceAccess(p)
					svc := serviceFromParams(p)
					ctx := p.Context
					sh := devicectx.SpruceHeaders(ctx)

					orgEntity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						RequestedInformation: &directory.RequestedInformation{
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS},
							Depth:             0,
						},
						Key: &directory.LookupEntitiesRequest_EntityID{
							EntityID: org.ID,
						},
						Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
						RootTypes:  []directory.EntityType{directory.EntityType_ORGANIZATION},
						ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL},
					})
					if err != nil {
						return nil, err
					}

					entities := make([]*models.Entity, 0, len(orgEntity.Members))
					for _, em := range orgEntity.Members {
						if em.Type == directory.EntityType_INTERNAL {
							ent, err := transformEntityToResponse(ctx, svc.staticURLPrefix, em, sh, gqlctx.Account(ctx))
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							entities = append(entities, ent)
						}
					}
					return entities, nil
				},
			},
			"savedThreadQueries": savedThreadQueriesField,
			"visitCategories":    visitCategoriesField,
			"deeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					svc := serviceFromParams(p)
					return deeplink.OrgURL(svc.webDomain, org.ID), nil
				},
			},
			"profile": &graphql.Field{
				Type: profileType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					ctx := p.Context
					ram := raccess.ResourceAccess(p)
					return lookupEntityProfile(ctx, ram, org.ID)
				},
			},
			"patientInviteURL": &graphql.Field{
				Type:              graphql.String,
				Resolve:           func(p graphql.ResolveParams) (interface{}, error) { return "", nil },
				DeprecationReason: "DEPRECATED due to practice links becoming plural per org. Use `practiceLinks` when it becomes available",
			},
			"partnerIntegrations": &graphql.Field{
				Type: graphql.NewList(partnerIntegrationType),
				Resolve: apiaccess.Authenticated(
					apiaccess.Provider(
						func(p graphql.ResolveParams) (interface{}, error) {
							org := p.Source.(*models.Organization)

							partnerIntegrations, err := lookupPartnerIntegrationsForOrg(p, org.ID)
							if err != nil {
								return nil, errors.InternalError(p.Context, err)
							}

							return partnerIntegrations, nil
						},
					)),
			},
			"savedMessageSections": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(savedMessageSectionType)),
				Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					return savedMessageSections(p, org.ID)
				}),
			},
			"supportThreadID": &graphql.Field{
				Type: graphql.String,
				Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					ctx := p.Context
					ram := raccess.ResourceAccess(p)
					acc := gqlctx.Account(ctx)

					ent, err := entityInOrgForAccountID(ctx, ram, org.ID, acc)
					if err != nil {
						return nil, err
					}

					queryThreadsRes, err := ram.QueryThreads(ctx, &threading.QueryThreadsRequest{
						ViewerEntityID: ent.ID,
						Iterator: &threading.Iterator{
							Direction: threading.ITERATOR_DIRECTION_FROM_START,
							Count:     2,
						},
						Type: threading.QUERY_THREADS_TYPE_ADHOC,
						QueryType: &threading.QueryThreadsRequest_Query{
							Query: &threading.Query{
								Expressions: []*threading.Expr{
									{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_SUPPORT}},
								},
							},
						},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					if len(queryThreadsRes.Edges) == 0 {
						// it is possible for the support thread for an organization to not exist
						return "", nil
					}

					// the support query brings up the support and setup assistant
					// so identify the support thread instead
					var supportThread *threading.Thread
					for _, threadEdge := range queryThreadsRes.Edges {
						if threadEdge.Thread.Type == threading.THREAD_TYPE_SUPPORT {
							supportThread = threadEdge.Thread
							break
						}
					}

					if supportThread != nil {
						return supportThread.ID, nil
					}

					return "", nil
				}),
			},
			"secureConversationCreationRequirement": &graphql.Field{
				Type:        graphql.NewNonNull(secureConversationCreationRequirementEnum),
				Description: "Requirement for creating a secure conversation",
				Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					svc := serviceFromParams(p)
					org := p.Source.(*models.Organization)

					twoFactorVerificationForSecureConversation, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
						NodeID: org.ID,
						Keys: []*settings.ConfigKey{
							{
								Key: invite.ConfigKeyTwoFactorVerificationForSecureConversation,
							},
						},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					if twoFactorVerificationForSecureConversation.Value {
						return secureConversationCreationRequirementPhoneAndEmail, nil
					}

					return secureConversationCreationRequirementPhoneOrEmail, nil
				}),
			},
		},
	},
)

func entityWithinOrg() func(p graphql.ResolveParams) (interface{}, error) {
	return apiaccess.Authenticated(
		func(p graphql.ResolveParams) (interface{}, error) {
			org := p.Source.(*models.Organization)
			if org.Entity != nil {
				return org.Entity, nil
			}

			ram := raccess.ResourceAccess(p)
			svc := serviceFromParams(p)
			ctx := p.Context
			acc := gqlctx.Account(ctx)

			e, err := entityInOrgForAccountID(ctx, ram, org.ID, acc)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if e == nil {
				return nil, errors.New("entity not found for organization")
			}
			sh := devicectx.SpruceHeaders(ctx)
			rE, err := transformEntityToResponse(ctx, svc.staticURLPrefix, e, sh, gqlctx.Account(ctx))
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			return rE, nil
		},
	)
}

func isSecureThreadsEnabled() func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		var orgID string
		switch s := p.Source.(type) {
		case *models.Organization:
			if s == nil {
				return false, nil
			}
			orgID = s.ID
		case *models.Thread:
			acc := gqlctx.Account(ctx)
			if s == nil || acc == nil || s.Type != models.ThreadTypeExternal || acc.Type != auth.AccountType_PROVIDER {
				return false, nil
			}
			orgID = s.OrganizationID
		default:
			golog.Errorf("Unhandled source type %T for isSecureThreadsEnabled, returning false", s)
			return false, nil
		}
		booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
			NodeID: orgID,
			Keys: []*settings.ConfigKey{
				{
					Key: baymaxgraphqlsettings.ConfigKeyCreateSecureThread,
				},
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		return booleanValue.Value, nil
	}
}
