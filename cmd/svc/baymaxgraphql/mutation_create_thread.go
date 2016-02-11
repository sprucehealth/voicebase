package main

import (
	"strings"

	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var createThreadInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateThreadInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId":     newClientMutationIDInputField(),
		"uuid":                 &graphql.InputObjectFieldConfig{Type: graphql.String},
		"organizationID":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"entityInfo":           &graphql.InputObjectFieldConfig{Type: entityInfoInputType},
		"createForContactInfo": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(contactInfoInputType)},
	},
})

const (
	createThreadErrorCodeExistingThread = "EXISTING_THREAD"
)

var createThreadErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "CreateThreadErrorCode",
	Description: "Result of createThread mutation",
	Values: graphql.EnumValueConfigMap{
		createThreadErrorCodeExistingThread: &graphql.EnumValueConfig{
			Value:       createThreadErrorCodeExistingThread,
			Description: "A thread exists with the provided contact",
		},
	},
})

type createThreadOutput struct {
	ClientMutationID string    `json:"clientMutationId,omitempty"`
	Success          bool      `json:"success"`
	ErrorCode        string    `json:"errorCode,omitempty"`
	ErrorMessage     string    `json:"errorMessage,omitempty"`
	Thread           *thread   `json:"thread"`
	ExistingThreads  []*thread `json:"existingThreads,omitempty"`
	NameDiffers      bool      `json:"nameDiffers"`
}

var createThreadOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreateThreadPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: createThreadErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"thread": &graphql.Field{
			Type:        graphql.NewNonNull(threadType),
			Description: "Populated for both SUCCESS and EXISTING_THREAD. For existing thread the server picks the most appropriate one if multiple.",
		},
		"existingThreads": &graphql.Field{
			Type:        graphql.NewList(graphql.NewNonNull(threadType)),
			Description: "Only for EXISTING_THREAD",
		},
		"nameDiffers": &graphql.Field{
			Type:        graphql.Boolean,
			Description: "Only for EXISTING_THREAD",
		},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*createThreadOutput)
		return ok
	},
})

var createThreadMutation = &graphql.Field{
	Type: graphql.NewNonNull(createThreadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createThreadInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		uuid, _ := input["uuid"].(string)
		orgID := input["organizationID"].(string)
		var entityInfo *directory.EntityInfo
		var contactInfos []*directory.Contact
		if ei, ok := input["entityInfo"].(map[string]interface{}); ok && ei != nil {
			var err error
			entityInfoInput := ei
			entityInfo, err = entityInfoFromInput(entityInfoInput)
			if err != nil {
				return nil, internalError(ctx, err)
			}
			contactInfos, err = contactListFromInput(entityInfoInput["contactInfos"].([]interface{}), true)
			if err != nil {
				return nil, internalError(ctx, err)
			}
		} else {
			entityInfo = &directory.EntityInfo{}
		}

		var err error
		entityInfo.DisplayName, err = buildDisplayName(entityInfo, contactInfos)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		createForContact, err := contactFromInput(input["createForContactInfo"])
		if err != nil {
			return nil, internalError(ctx, err)
		}

		creatorEnt, err := svc.entityForAccountID(ctx, orgID, acc.ID)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		if creatorEnt == nil {
			return nil, errors.New("Not a member of the organization")
		}

		// Check for an existing entity with the provided contact info
		var existingEntities []*directory.Entity
		dres, err := svc.directory.LookupEntitiesByContact(ctx, &directory.LookupEntitiesByContactRequest{
			ContactValue: createForContact.Value,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             1,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
		})
		if err == nil {
			// Filter out entities that aren't external as that's all we're dealing with right now
			existingEntities = make([]*directory.Entity, 0, len(dres.Entities))
			for _, e := range dres.Entities {
				if e.Type == directory.EntityType_EXTERNAL {
					// Make sure entity is a member of the chosen organization
					for _, em := range e.Memberships {
						if em.ID == orgID {
							existingEntities = append(existingEntities, e)
							break
						}
					}
				}
			}
		} else if err != nil && grpc.Code(err) != codes.NotFound {
			return nil, internalError(ctx, err)
		}

		// Check for an existing thread
		if len(existingEntities) != 0 {
			threadResults := make([][]*threading.Thread, len(existingEntities))
			par := conc.NewParallel()
			for i, e := range existingEntities {
				ix := i
				ent := e
				par.Go(func() error {
					res, err := svc.threading.ThreadsForMember(ctx, &threading.ThreadsForMemberRequest{
						EntityID:    ent.ID,
						PrimaryOnly: true,
					})
					if err != nil {
						if grpc.Code(err) != codes.NotFound {
							return err
						}
						return nil
					}
					threadResults[ix] = res.Threads
					return nil
				})
			}
			if err := par.Wait(); err != nil {
				return nil, internalError(ctx, err)
			}
			var threads []*threading.Thread
			for _, ts := range threadResults {
				threads = append(threads, ts...)
			}

			if len(threads) != 0 {
				// See if there's an existing entity with a matching first and last name. This isn't
				// necessarily a strong match, but this whole process is best effort.
				var matchingEntity *directory.Entity
				for _, e := range existingEntities {
					if strings.EqualFold(e.Info.FirstName, entityInfo.FirstName) && strings.EqualFold(e.Info.LastName, entityInfo.LastName) {
						matchingEntity = e
						break
					}
				}

				// Build a map of entities by ID so later we can lookup primary entity efficiently to generate thread tiel
				entMap := make(map[string]*directory.Entity, len(existingEntities))
				for _, e := range existingEntities {
					entMap[e.ID] = e
				}

				var theOneThread *thread
				var matchedName bool
				existingThreads := make([]*thread, len(threads))
				for i, t := range threads {
					// Sanity check. This shouldn't ever be triggered since we made sure the tntiy
					// is part of the organization, but doesn't hurt to double check.
					if t.OrganizationID != orgID {
						golog.Errorf("Thread %s not part of organization %s but entity %s is", t.OrganizationID, orgID, t.PrimaryEntityID)
						continue
					}
					th, err := transformThreadToResponse(t)
					if err != nil {
						return nil, internalError(ctx, err)
					}
					th.Title = threadTitleForEntity(entMap[t.PrimaryEntityID])

					existingThreads[i] = th
					// See if there's a thread with a primary entity equal to the one we foudn matching the contact info
					if matchingEntity != nil && th.PrimaryEntityID == matchingEntity.ID {
						theOneThread = th
						matchedName = true
					}
				}

				if theOneThread == nil {
					// If we didn't exactly match a thread by contact info then pick the latest
					theOneThread = existingThreads[0]
					for _, t := range existingThreads[1:] {
						if t.LastMessageTimestamp > theOneThread.LastMessageTimestamp {
							theOneThread = t
						}
					}
				}

				return &createThreadOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        createThreadErrorCodeExistingThread,
					ErrorMessage:     "A thread already exists with the provided contact",
					Thread:           theOneThread,
					ExistingThreads:  existingThreads,
					NameDiffers:      !matchedName,
				}, nil
			}
		}

		// Sort contactsInfos to put the 'createForContact' at the top as that's the implicit
		// signifier for primary. This is important when the new entity is created so that the
		// default channel for sending messages becomes the requested one.
		hasContact := false
		for i, c := range contactInfos {
			if c.ContactType == createForContact.ContactType && c.Value == createForContact.Value {
				if i != 0 {
					contactInfos[0], contactInfos[i] = contactInfos[i], contactInfos[0]
				}
				hasContact = true
				break
			}
		}
		// If the contacts list didn't have the contact info for the thread then add it
		if !hasContact {
			contactInfos = append([]*directory.Contact{createForContact}, contactInfos...)
		}

		// Didn't find any existing threads so create a new one, but first we need to create an entity. We
		// purposefully don't try to merge with an existing entity even if some contact info matches since
		// that's likely very error prone. Safer to just assume a new entity.
		ceres, err := svc.directory.CreateEntity(ctx, &directory.CreateEntityRequest{
			Type: directory.EntityType_EXTERNAL,
			InitialMembershipEntityID: orgID,
			Contacts:                  contactInfos,
			EntityInfo:                entityInfo,
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
			},
		})
		if err != nil {
			return nil, internalError(ctx, err)
		}
		primaryEnt := ceres.Entity

		res, err := svc.threading.CreateEmptyThread(ctx, &threading.CreateEmptyThreadRequest{
			UUID:           uuid,
			OrganizationID: orgID,
			FromEntityID:   creatorEnt.ID,
			Source: &threading.Endpoint{
				Channel: threading.Endpoint_APP,
				ID:      creatorEnt.ID,
			},
			PrimaryEntityID: primaryEnt.ID,
			Summary:         "New conversation", // TODO: not sure what we want here. it's a fallback if there's no posts made in the thread.
		})
		if err != nil {
			return nil, internalError(ctx, err)
		}
		th, err := transformThreadToResponse(res.Thread)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		th.Title = threadTitleForEntity(primaryEnt)

		return &createThreadOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Thread:           th,
		}, nil
	},
}
