package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var threadConnectionType = ConnectionDefinitions(ConnectionConfig{
	Name:     "Thread",
	NodeType: threadType,
})

var threadTypeIndicatorEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "ThreadTypeIndicator",
	Values: graphql.EnumValueConfigMap{
		models.ThreadTypeIndicatorNone: &graphql.EnumValueConfig{
			Value:       models.ThreadTypeIndicatorNone,
			Description: "No indicator is provided for this thread type",
		},
		models.ThreadTypeIndicatorLock: &graphql.EnumValueConfig{
			Value:       models.ThreadTypeIndicatorLock,
			Description: "Describes that the thread can be described with the lock indicator",
		},
		models.ThreadTypeIndicatorGroup: &graphql.EnumValueConfig{
			Value:       models.ThreadTypeIndicatorGroup,
			Description: "Describes that the thread can be described with the group indicator",
		},
	},
})

var threadType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Thread",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":                    &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"typeIndicator":         &graphql.Field{Type: graphql.NewNonNull(threadTypeIndicatorEnum)},
			"allowAddMembers":       &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowDelete":           &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowVideoAttachments": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowEmailAttachments": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"isPatientThread":       &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"isTeamThread":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowVisitAttachments": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					ram := raccess.ResourceAccess(p)
					th := p.Source.(*models.Thread)
					acc := gqlctx.Account(p.Context)
					if acc == nil {
						return false, errors.ErrNotAuthenticated(ctx)
					}

					if th.Type != models.ThreadTypeSecureExternal {
						return false, nil
					}

					if acc.Type != auth.AccountType_PROVIDER {
						return false, nil
					}

					booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
						NodeID: th.OrganizationID,
						Keys: []*settings.ConfigKey{
							{
								Key: baymaxgraphqlsettings.ConfigKeyVisitAttachments,
							},
						},
					})
					if err != nil {
						return false, errors.InternalError(ctx, err)
					}

					if !booleanValue.Value {
						return false, nil
					}
					// only allow visit attachments if the patient has created an account and is on iOS
					primaryEntity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
						LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
							EntityID: th.PrimaryEntityID,
						},
						Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
						RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
					})
					if err != nil {
						if grpc.Code(err) == codes.NotFound {
							return false, nil
						}
						return nil, errors.InternalError(ctx, err)
					}

					// if patient has not created account then we cannot send spruce visit just yet
					if primaryEntity.AccountID == "" {
						return false, nil
					}

					loginInfoRes, err := ram.LastLoginForAccount(ctx, &auth.GetLastLoginInfoRequest{
						AccountID: primaryEntity.AccountID,
					})
					if err != nil {
						if grpc.Code(err) == codes.NotFound {
							// we dont have login information for the patient yet
							return false, nil
						}
						return false, errors.InternalError(ctx, err)
					}
					return loginInfoRes.Platform == auth.Platform_IOS, nil
				},
			},
			"allowCarePlanAttachments": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					th := p.Source.(*models.Thread)
					acc := gqlctx.Account(p.Context)
					if acc == nil {
						return false, errors.ErrNotAuthenticated(ctx)
					}

					if th.Type != models.ThreadTypeSecureExternal {
						return false, nil
					}

					if acc.Type != auth.AccountType_PROVIDER {
						return false, nil
					}

					booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
						NodeID: th.OrganizationID,
						Keys: []*settings.ConfigKey{
							{
								Key: baymaxgraphqlsettings.ConfigKeyCarePlans,
							},
						},
					})
					if err != nil {
						return false, errors.InternalError(ctx, err)
					}
					return booleanValue.Value, nil
				},
			},
			"allowExternalDelivery": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowInternalMessages": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowLeave":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowMentions":         &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowRemoveMembers":    &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowSMSAttachments":   &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowUpdateTitle":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowInvitePatientToSecureThread": &graphql.Field{
				Type:    graphql.NewNonNull(graphql.Boolean),
				Resolve: isSecureThreadsEnabled(),
			},
			"callableIdentities": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(callableIdentityType)),
				Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					ram := raccess.ResourceAccess(p)
					svc := serviceFromParams(p)
					acc := gqlctx.Account(p.Context)
					th := p.Source.(*models.Thread)
					switch th.Type {
					case models.ThreadTypeSecureExternal, models.ThreadTypeExternal:
					case models.ThreadTypeTeam:
						if environment.IsProd() {
							return []*models.CallableIdentity{}, nil
						}
						memberEntities, err := ram.ThreadMembers(ctx, th.OrganizationID, &threading.ThreadMembersRequest{
							ThreadID: th.ID,
						})
						if err != nil {
							return nil, err
						}
						dh := devicectx.SpruceHeaders(ctx)
						idents := make([]*models.CallableIdentity, len(memberEntities))
						for i, e := range memberEntities {
							endpoints, err := callableEndpointsForEntity(e)
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							ent, err := transformEntityToResponse(svc.staticURLPrefix, e, dh, acc)
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							idents[i] = &models.CallableIdentity{
								Name:      e.Info.DisplayName,
								Endpoints: endpoints,
								Entity:    ent,
							}
						}
						return idents, nil
					default:
						return []*models.CallableIdentity{}, nil
					}
					if th.PrimaryEntityID == "" {
						return []*models.CallableIdentity{}, nil
					}
					if acc.Type != auth.AccountType_PROVIDER {
						return []*models.CallableIdentity{}, nil
					}
					ent, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
						LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
							EntityID: th.PrimaryEntityID,
						},
						RequestedInformation: &directory.RequestedInformation{
							Depth:             0,
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
						},
						Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
						RootTypes: []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					if ent == nil {
						return []*models.CallableIdentity{}, nil
					}
					endpoints, err := callableEndpointsForEntity(ent)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					ment, err := transformEntityToResponse(svc.staticURLPrefix, ent, devicectx.SpruceHeaders(ctx), acc)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					return []*models.CallableIdentity{{
						Name:      ent.Info.DisplayName,
						Endpoints: endpoints,
						Entity:    ment,
					}}, nil
				}),
			},
			"emptyStateTextMarkup": &graphql.Field{Type: graphql.String},
			"lastMessageTimestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"subtitle":             &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"title":                &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"unread":               &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"unreadReference":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"isDeletable": &graphql.Field{
				Type:              graphql.NewNonNull(graphql.Boolean),
				DeprecationReason: "Replaced with allowDelete",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*models.Thread).AllowDelete, nil
				},
			},
			"members": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(entityType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					// Only team threads have members
					if th.Type != models.ThreadTypeTeam {
						return nil, nil
					}

					svc := serviceFromParams(p)
					acc := gqlctx.Account(p.Context)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}
					ram := raccess.ResourceAccess(p)
					members, err := ram.ThreadMembers(ctx, th.OrganizationID, &threading.ThreadMembersRequest{
						ThreadID: th.ID,
					})
					if err != nil {
						return nil, err
					}
					sh := devicectx.SpruceHeaders(ctx)
					ms := make([]*models.Entity, len(members))
					for i, em := range members {
						e, err := transformEntityToResponse(svc.staticURLPrefix, em, sh, acc)
						if err != nil {
							return nil, err
						}
						ms[i] = e
					}
					return ms, nil
				},
			},
			"addressableEntities": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(entityType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}

					svc := serviceFromParams(p)
					acc := gqlctx.Account(p.Context)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}
					ram := raccess.ResourceAccess(p)

					switch th.Type {
					case models.ThreadTypeTeam:
						members, err := ram.ThreadMembers(ctx, th.OrganizationID, &threading.ThreadMembersRequest{
							ThreadID: th.ID,
						})
						if err != nil {
							return nil, err
						}
						ms := make([]*models.Entity, len(members))
						for i, em := range members {
							e, err := transformEntityToResponse(svc.staticURLPrefix, em, devicectx.SpruceHeaders(ctx), acc)
							if err != nil {
								return nil, err
							}
							ms[i] = e
						}
						return ms, nil
					case models.ThreadTypeExternal, models.ThreadTypeSupport, models.ThreadTypeLegacyTeam, models.ThreadTypeSecureExternal:

						// no addressable entities to return for a support thread not in spruce support
						if th.Type == models.ThreadTypeSupport && th.OrganizationID != *flagSpruceOrgID {
							return nil, nil
						}

						orgEntity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: th.OrganizationID,
							},
							RequestedInformation: &directory.RequestedInformation{
								Depth:             0,
								EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS},
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
								ent, err := transformEntityToResponse(svc.staticURLPrefix, em, devicectx.SpruceHeaders(ctx), acc)
								if err != nil {
									return nil, errors.InternalError(ctx, err)
								}
								entities = append(entities, ent)
							}
						}
						return entities, nil
					}
					return nil, nil
				},
			},
			// TODO: We currently just assume all contacts for an entity are available endpoints
			"availableEndpoints": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(endpointType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					// No endpoints for team threads
					if th.Type == models.ThreadTypeTeam || th.Type == models.ThreadTypeSecureExternal {
						return nil, nil
					}

					ram := raccess.ResourceAccess(p)
					ent, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
						LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
							EntityID: th.PrimaryEntityID,
						},
						RequestedInformation: &directory.RequestedInformation{
							Depth:             0,
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
						},
						Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					})
					if err != nil {
						return nil, err
					}
					if ent.Type != directory.EntityType_EXTERNAL {
						return []*models.Endpoint{}, nil
					}

					endpoints := make([]*models.Endpoint, len(ent.Contacts))
					for i, c := range ent.Contacts {
						endpoint, err := transformEntityContactToEndpoint(c)
						if err != nil {
							return nil, errors.InternalError(ctx, err)
						}
						endpoints[i] = endpoint
					}
					return endpoints, nil
				},
			},
			// Default endpoints are build from the last primary entity endpoints filtering out anything contacts that no longer exist for the entity
			"defaultEndpoints": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(endpointType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					// No endpoints for team threads
					if th.Type == models.ThreadTypeTeam || th.Type == models.ThreadTypeSecureExternal {
						return nil, nil
					}

					ram := raccess.ResourceAccess(p)

					ent, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
						LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
							EntityID: th.PrimaryEntityID,
						},
						RequestedInformation: &directory.RequestedInformation{
							Depth:             0,
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
						},
						Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					})
					if err != nil {
						return nil, err
					}
					if ent.Type != directory.EntityType_EXTERNAL {
						return []*models.Endpoint{}, nil
					}

					var filteredEndpoints []*models.Endpoint
					// Assert that our endpoints still exist as a contact
					for _, ep := range th.LastPrimaryEntityEndpoints {
						for _, c := range ent.Contacts {
							endpoint, err := transformEntityContactToEndpoint(c)
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							if endpoint.Channel == ep.Channel && endpoint.ID == ep.ID {
								filteredEndpoints = append(filteredEndpoints, endpoint)
								break
							}
						}
					}
					// If we didn't find any matching endpoints or the source list is empty, pick the first contact attached to the entity
					if len(filteredEndpoints) == 0 {
						for _, c := range ent.Contacts {
							endpoint, err := transformEntityContactToEndpoint(c)
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							filteredEndpoints = append(filteredEndpoints, endpoint)
							break
						}
					}
					return filteredEndpoints, nil
				},
			},
			"primaryEntity": &graphql.Field{
				Type: entityType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					svc := serviceFromParams(p)
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					// Internal threads don't have a primary entity
					if th.PrimaryEntityID == "" {
						// TODO: for now returning a stub primary entity as apps are relying on it existing. remove at some point
						return stubEntity, nil
					}
					if selectingOnlyID(p) {
						return &models.Entity{ID: th.PrimaryEntityID}, nil
					}

					ram := raccess.ResourceAccess(p)
					pe, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
						LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
						LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
							EntityID: th.PrimaryEntityID,
						},
						RequestedInformation: &directory.RequestedInformation{
							Depth:             0,
							EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
						},
						Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					})
					if err != nil {
						return nil, err
					}
					sh := devicectx.SpruceHeaders(ctx)
					ent, err := transformEntityToResponse(svc.staticURLPrefix, pe, sh, gqlctx.Account(ctx))
					if err != nil {
						return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform entity: %s", err))
					}
					return ent, nil
				},
			},
			"items": &graphql.Field{
				Type: graphql.NewNonNull(threadItemConnectionType.ConnectionType),
				Args: NewConnectionArguments(nil),
				Resolve: apiaccess.Authenticated(
					func(p graphql.ResolveParams) (interface{}, error) {
						ctx := p.Context
						t := p.Source.(*models.Thread)
						if t == nil {
							return nil, errors.InternalError(ctx, errors.New("thread is nil"))
						}
						svc := serviceFromParams(p)
						ram := raccess.ResourceAccess(p)
						acc := gqlctx.Account(p.Context)

						ent, err := entityInOrgForAccountID(ctx, ram, t.OrganizationID, acc)
						if err != nil {
							return nil, errors.InternalError(ctx, err)
						}

						req := &threading.ThreadItemsRequest{
							ThreadID:       t.ID,
							ViewerEntityID: ent.ID,
							Iterator:       &threading.Iterator{},
						}
						if s, ok := p.Args["after"].(string); ok {
							req.Iterator.StartCursor = s
						}
						if s, ok := p.Args["before"].(string); ok {
							req.Iterator.EndCursor = s
						}
						if i, ok := p.Args["last"].(int); ok {
							req.Iterator.Count = uint32(i)
							req.Iterator.Direction = threading.Iterator_FROM_END
						} else if i, ok := p.Args["first"].(int); ok {
							req.Iterator.Count = uint32(i)
							req.Iterator.Direction = threading.Iterator_FROM_START
						} else {
							req.Iterator.Count = 20 // default
							req.Iterator.Direction = threading.Iterator_FROM_START
						}
						res, err := ram.ThreadItems(ctx, req)
						if err != nil {
							return nil, err
						}

						cn := &Connection{
							Edges: make([]*Edge, len(res.Edges)),
						}
						if req.Iterator.Direction == threading.Iterator_FROM_START {
							cn.PageInfo.HasNextPage = res.HasMore
						} else {
							cn.PageInfo.HasPreviousPage = res.HasMore
						}

						for i, e := range res.Edges {
							it, err := transformThreadItemToResponse(e.Item, "", acc.ID, svc.webDomain, svc.mediaAPIDomain)
							if err != nil {
								golog.Errorf("Failed to transform thread item %s: %s", e.Item.ID, err)
								continue
							}
							cn.Edges[i] = &Edge{
								Node:   it,
								Cursor: ConnectionCursor(e.Cursor),
							}
						}

						return cn, nil
					}),
			},
			"deeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Args: graphql.FieldConfigArgument{
					"savedQueryID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					th := p.Source.(*models.Thread)
					svc := serviceFromParams(p)
					savedQueryID, _ := p.Args["savedQueryID"].(string)
					return deeplink.ThreadURL(svc.webDomain, th.OrganizationID, savedQueryID, th.ID), nil
				},
			},
			"shareableDeeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					th := p.Source.(*models.Thread)
					svc := serviceFromParams(p)
					return deeplink.ThreadURLShareable(svc.webDomain, th.OrganizationID, th.ID), nil
				},
			},
		},
	},
)

func lookupThread(ctx context.Context, ram raccess.ResourceAccessor, threadID, viewerEntityID string) (*models.Thread, error) {
	thread, err := ram.Thread(ctx, threadID, viewerEntityID)
	if err != nil {
		return nil, err
	}

	th, err := transformThreadToResponse(ctx, ram, thread, gqlctx.Account(ctx))
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	if err := hydrateThreads(ctx, ram, []*models.Thread{th}); err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return th, nil
}
