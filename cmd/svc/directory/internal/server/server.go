package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	// ErrNotImplemented is returned from RPC calls that have yet to be implemented
	ErrNotImplemented = errors.New("Not Implemented")
)

type server struct {
	dl                         dal.DAL
	statLookupEntitiesEntities *metrics.Counter
	publisher                  events.Publisher
}

// New returns an initialized instance of server
func New(dl dal.DAL, publisher events.Publisher, metricsRegistry metrics.Registry) directory.DirectoryServer {
	srv := &server{
		dl: dl,
		statLookupEntitiesEntities: metrics.NewCounter(),
		publisher:                  publisher,
	}
	metricsRegistry.Add("LookupEntities.entities", srv.statLookupEntitiesEntities)
	return srv
}

// This is a helper to avoid duplicating this nil check everywhere
func riEntityInformation(ri *directory.RequestedInformation) []directory.EntityInformation {
	if ri == nil {
		return nil
	}
	return ri.EntityInformation
}

// This is a helper to avoid duplicating this nil check everywhere
func riDepth(ri *directory.RequestedInformation) int64 {
	if ri == nil {
		return 0
	}
	return ri.Depth
}

func appendMembershipEntityInformation(entityInfo []directory.EntityInformation) []directory.EntityInformation {
	if entityInfo == nil {
		return []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS}
	}
	for _, ei := range entityInfo {
		if ei == directory.EntityInformation_MEMBERSHIPS {
			return entityInfo
		}
	}
	return append(entityInfo, directory.EntityInformation_MEMBERSHIPS)
}

func filterToMembersOnly(pbEntities []*directory.Entity, memberOfEntity string) []*directory.Entity {
	if memberOfEntity == "" {
		return pbEntities
	}

	filteredpbEntities := make([]*directory.Entity, 0, len(pbEntities))
	for _, pbEntity := range pbEntities {
		for _, m := range pbEntity.Memberships {
			if m.ID == memberOfEntity {
				filteredpbEntities = append(filteredpbEntities, pbEntity)
				break
			}
		}
	}
	return filteredpbEntities
}

func (s *server) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest) (out *directory.LookupEntitiesResponse, err error) {
	var entityIDs []dal.EntityID
	switch key := in.LookupKeyOneof.(type) {
	case *directory.LookupEntitiesRequest_AccountID:
		// TODO: Actually use the account_id field on the table and don't do this double lookup
		externalEntityIDs, err := s.dl.ExternalEntityIDs(key.AccountID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		entityIDs = make([]dal.EntityID, len(externalEntityIDs))
		for i, v := range externalEntityIDs {
			entityIDs[i] = v.EntityID
		}
	case *directory.LookupEntitiesRequest_ExternalID:
		externalEntityIDs, err := s.dl.ExternalEntityIDs(key.ExternalID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		entityIDs = make([]dal.EntityID, len(externalEntityIDs))
		for i, v := range externalEntityIDs {
			entityIDs[i] = v.EntityID
		}
	case *directory.LookupEntitiesRequest_EntityID:
		entityID, err := dal.ParseEntityID(key.EntityID)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id %q", key.EntityID)
		}
		entityIDs = []dal.EntityID{entityID}
	case *directory.LookupEntitiesRequest_BatchEntityID:
		entityIDs = make([]dal.EntityID, len(key.BatchEntityID.IDs))
		for i, id := range key.BatchEntityID.IDs {
			eid, err := dal.ParseEntityID(id)
			if err != nil {
				return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id %q", id)
			}
			entityIDs[i] = eid
		}
	default:
		return nil, errors.Errorf("Unknown lookup key type %T", in.LookupKeyOneof)
	}
	statuses, err := transformEntityStatuses(in.Statuses)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}

	rootTypes, err := transformEntityTypes(in.RootTypes)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}

	childTypes, err := transformEntityTypes(in.ChildTypes)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}

	entities, err := s.dl.Entities(entityIDs, statuses, rootTypes)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(entities) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "No entities located matching query")
	}

	if in.MemberOfEntity != "" {
		// ensure that memberships is in the requested information
		if in.RequestedInformation == nil {
			in.RequestedInformation = &directory.RequestedInformation{}
		}

		in.RequestedInformation.EntityInformation = appendMembershipEntityInformation(in.RequestedInformation.EntityInformation)
	}

	pbEntities, err := getPBEntities(s.dl, entities, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), statuses, childTypes)
	if err != nil {
		return nil, errors.Trace(err)
	}

	s.statLookupEntitiesEntities.Inc(uint64(len(pbEntities)))
	return &directory.LookupEntitiesResponse{
		Entities: filterToMembersOnly(pbEntities, in.MemberOfEntity),
	}, nil
}

func (s *server) CreateEntity(ctx context.Context, in *directory.CreateEntityRequest) (out *directory.CreateEntityResponse, err error) {
	if err := s.validateCreateEntityRequest(in); err != nil {
		return nil, err
	}

	entityType, err := dal.ParseEntityType(directory.EntityType_name[int32(in.Type)])
	if err != nil {
		return nil, errors.Trace(err)
	}
	var pbEntity *directory.Entity
	if err = s.dl.Transact(func(dl dal.DAL) error {
		var entityGender *dal.EntityGender
		if in.EntityInfo.Gender != directory.EntityInfo_UNKNOWN {
			eg, err := dal.ParseEntityGender(in.EntityInfo.Gender.String())
			if err != nil {
				return errors.Trace(err)
			}
			entityGender = &eg
		}
		var dob *encoding.Date
		if in.EntityInfo.DOB != nil {
			dob = &encoding.Date{
				Month: int(in.EntityInfo.DOB.Month),
				Day:   int(in.EntityInfo.DOB.Day),
				Year:  int(in.EntityInfo.DOB.Year),
			}
		}

		var displayName string
		if in.EntityInfo.DisplayName != "" {
			displayName = in.EntityInfo.DisplayName
		} else {
			displayName = buildDisplayName(in.EntityInfo, in.Contacts)
			if len(displayName) == 0 {
				return errors.Trace(errors.New("Not enough information to build the display name for entity"))
			}
		}

		entityID, err := dl.InsertEntity(&dal.Entity{
			Type:          entityType,
			Status:        dal.EntityStatusActive,
			FirstName:     in.EntityInfo.FirstName,
			MiddleInitial: in.EntityInfo.MiddleInitial,
			LastName:      in.EntityInfo.LastName,
			GroupName:     in.EntityInfo.GroupName,
			DisplayName:   displayName,
			ShortTitle:    in.EntityInfo.ShortTitle,
			LongTitle:     in.EntityInfo.LongTitle,
			Gender:        entityGender,
			DOB:           dob,
			AccountID:     in.AccountID,
			Note:          in.EntityInfo.Note,
			Source:        directory.FlattenEntitySource(in.Source),
		})
		if err != nil {
			return errors.Trace(err)
		}
		if in.ExternalID != "" {
			if err := dl.InsertExternalEntityID(&dal.ExternalEntityID{
				EntityID:   entityID,
				ExternalID: in.ExternalID,
			}); err != nil {
				return errors.Trace(err)
			}
		}
		if in.InitialMembershipEntityID != "" {
			targetEntityID, err := dal.ParseEntityID(in.InitialMembershipEntityID)
			if err != nil {
				return errors.Trace(err)
			}
			if err := dl.InsertEntityMembership(&dal.EntityMembership{
				EntityID:       entityID,
				TargetEntityID: targetEntityID,
				Status:         dal.EntityMembershipStatusActive,
			}); err != nil {
				return errors.Trace(err)
			}
		}
		for _, contact := range in.Contacts {
			contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(contact.ContactType)])
			if err != nil {
				return errors.Trace(err)
			}
			if _, err := dl.InsertEntityContact(&dal.EntityContact{
				EntityID:    entityID,
				Type:        contactType,
				Value:       contact.Value,
				Provisioned: contact.Provisioned,
				Label:       contact.Label,
				Verified:    contact.Verified,
			}); err != nil {
				return errors.Trace(err)
			}
		}
		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), nil, nil)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.CreateEntityResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) CreateExternalIDs(ctx context.Context, in *directory.CreateExternalIDsRequest) (out *directory.CreateExternalIDsResponse, err error) {
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID cannot be empty")
	}
	if err = s.dl.Transact(func(dl dal.DAL) error {
		models := make([]*dal.ExternalEntityID, 0, len(in.ExternalIDs))
		for _, eID := range in.ExternalIDs {
			entityID, err := dal.ParseEntityID(in.EntityID)
			if err != nil {
				return grpc.Errorf(codes.InvalidArgument, err.Error())
			}
			models = append(models, &dal.ExternalEntityID{EntityID: entityID, ExternalID: eID})
		}
		if len(models) != 0 {
			if err := dl.InsertExternalEntityIDs(models); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &directory.CreateExternalIDsResponse{}, nil
}

func (s *server) SerializedEntityContact(ctx context.Context, in *directory.SerializedEntityContactRequest) (out *directory.SerializedEntityContactResponse, err error) {
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "entity id required")
	}
	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Error parsing entity id: %s", err)
	}
	platform, err := dal.ParseSerializedClientEntityContactPlatform(directory.Platform_name[int32(in.Platform)])
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Error parsing platform type: %s", err)
	}
	sec, err := s.dl.SerializedClientEntityContact(entityID, platform)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "No serialized entity contact exists for entity id %s and platform %s", entityID, platform)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.SerializedEntityContactResponse{
		SerializedEntityContact: transformSerializedClientEntityContactToResponse(sec),
	}, nil
}

func (s *server) validateCreateEntityRequest(in *directory.CreateEntityRequest) error {
	if in.InitialMembershipEntityID != "" {
		eID, err := dal.ParseEntityID(in.InitialMembershipEntityID)
		if err != nil {
			return grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id")
		}
		exists, err := doesEntityExist(s.dl, eID)
		if err != nil {
			return errors.Trace(err)
		}
		if !exists {
			return grpc.Errorf(codes.NotFound, "Entity not found %s", in.InitialMembershipEntityID)
		}
	}
	if in.EntityInfo == nil {
		in.EntityInfo = &directory.EntityInfo{}
	}
	for _, contact := range in.Contacts {
		if contact.Value == "" {
			return grpc.Errorf(codes.InvalidArgument, "Contact value cannot be empty")
		}
		if err := validateContact(contact); err != nil {
			return grpc.Errorf(codes.InvalidArgument, err.Error())
		}
	}
	return nil
}

func (s *server) UpdateEntity(ctx context.Context, in *directory.UpdateEntityRequest) (out *directory.UpdateEntityResponse, err error) {
	oldEnt, err := s.validateUpdateEntityRequest(in)
	if err != nil {
		return nil, err
	}
	defer oldEnt.Recycle()

	eID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity ID")
	}

	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		entityUpdate := &dal.EntityUpdate{}

		entityInfo := transformEntityToEntityInfoResponse(oldEnt)
		if in.UpdateEntityInfo {
			entityInfo = in.EntityInfo
			entityUpdate.FirstName = &in.EntityInfo.FirstName
			entityUpdate.MiddleInitial = &in.EntityInfo.MiddleInitial
			entityUpdate.LastName = &in.EntityInfo.LastName
			entityUpdate.GroupName = &in.EntityInfo.GroupName
			entityUpdate.ShortTitle = &in.EntityInfo.ShortTitle
			entityUpdate.LongTitle = &in.EntityInfo.LongTitle
			entityUpdate.Note = &in.EntityInfo.Note

			if in.EntityInfo.Gender != directory.EntityInfo_UNKNOWN {
				g, err := dal.ParseEntityGender(in.EntityInfo.Gender.String())
				if err != nil {
					return grpc.Errorf(codes.InvalidArgument, "Unknown entity gender %s", in.EntityInfo.Gender.String())
				}
				entityUpdate.Gender = &g
			}
			if in.EntityInfo.DOB != nil {
				entityUpdate.DOB = &encoding.Date{
					Month: int(in.EntityInfo.DOB.Month),
					Day:   int(in.EntityInfo.DOB.Day),
					Year:  int(in.EntityInfo.DOB.Year),
				}
			}
		}

		if in.UpdateImageMediaID {
			entityUpdate.ImageMediaID = &in.ImageMediaID
		}

		if in.UpdateAccountID {
			entityUpdate.AccountID = &in.AccountID
		}

		var contacts []*directory.Contact
		if in.UpdateContacts {
			// Delete existing contact info
			if _, err := dl.DeleteEntityContactsForEntityID(eID); err != nil {
				return errors.Trace(err)
			}

			// Insert the new set of contacts
			if len(in.Contacts) != 0 {
				dalContacts := make([]*dal.EntityContact, len(in.Contacts))
				for i, contact := range in.Contacts {
					contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(contact.ContactType)])
					if err != nil {
						return errors.Trace(err)
					}
					dalContacts[i] = &dal.EntityContact{
						EntityID:    eID,
						Type:        contactType,
						Value:       contact.Value,
						Provisioned: contact.Provisioned,
						Label:       contact.Label,
						Verified:    contact.Verified,
					}
				}
				if err := dl.InsertEntityContacts(dalContacts); err != nil {
					return errors.Trace(err)
				}

				contacts = in.Contacts
			}
		} else {
			// For external entities need to fetch the contacts to build the display name
			if oldEnt.Type == dal.EntityTypeExternal {
				cs, err := dl.EntityContacts(eID)
				if err != nil {
					return errors.Trace(err)
				}
				contacts = make([]*directory.Contact, len(cs))
				for i, c := range cs {
					contacts[i] = transformEntityContactToResponse(c)
				}
			}
		}

		if dp := buildDisplayName(entityInfo, contacts); dp != "" {
			entityUpdate.DisplayName = &dp
		}

		if _, err := dl.UpdateEntity(eID, entityUpdate); err != nil {
			return errors.Trace(err)
		}

		// Upsert any serialized entity contact info
		if in.UpdateSerializedEntityContacts {
			for _, sec := range in.SerializedEntityContacts {
				platform, err := dal.ParseSerializedClientEntityContactPlatform(directory.Platform_name[int32(sec.Platform)])
				if err != nil {
					return grpc.Errorf(codes.InvalidArgument, "Error parsing platform type: %s", err)
				}
				if err := dl.UpsertSerializedClientEntityContact(&dal.SerializedClientEntityContact{
					EntityID:                eID,
					Platform:                platform,
					SerializedEntityContact: sec.SerializedEntityContact,
				}); err != nil {
					return errors.Trace(err)
				}
			}
		}

		entity, err := dl.Entity(eID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), nil, nil)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}

	s.publisher.PublishAsync(&directory.EntityUpdatedEvent{
		EntityID: in.EntityID,
	})

	return &directory.UpdateEntityResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) validateUpdateEntityRequest(in *directory.UpdateEntityRequest) (*dal.Entity, error) {
	eID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity ID")
	}
	ent, err := s.dl.Entity(eID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, err.Error())
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return ent, nil
}

func (s *server) ExternalIDs(ctx context.Context, in *directory.ExternalIDsRequest) (out *directory.ExternalIDsResponse, err error) {
	ids := make([]dal.EntityID, len(in.EntityIDs))
	for i, id := range in.EntityIDs {
		eID, err := dal.ParseEntityID(id)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id")
		}
		ids[i] = eID
	}
	externalIDs, err := s.dl.ExternalEntityIDsForEntities(ids)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.ExternalIDsResponse{
		ExternalIDs: transformExternalIDs(externalIDs),
	}, nil
}

func doesEntityExist(dl dal.DAL, entityID dal.EntityID) (bool, error) {
	golog.Debugf("Entering server.doesEntityExist: %s", entityID)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.doesEntityExist...") }()
	}
	_, err := dl.Entity(entityID)
	if errors.Cause(err) == dal.ErrNotFound {
		return false, nil
	}
	return true, errors.Trace(err)
}

func (s *server) CreateMembership(ctx context.Context, in *directory.CreateMembershipRequest) (*directory.CreateMembershipResponse, error) {
	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id")
	}
	targetEntityID, err := dal.ParseEntityID(in.TargetEntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id")
	}
	exists, err := doesEntityExist(s.dl, entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if !exists {
		return nil, grpc.Errorf(codes.NotFound, "Entity not found %s", in.EntityID)
	}
	exists, err = doesEntityExist(s.dl, targetEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if !exists {
		return nil, grpc.Errorf(codes.NotFound, "Entity not found %s", in.TargetEntityID)
	}

	if err := s.dl.InsertEntityMembership(&dal.EntityMembership{
		EntityID:       entityID,
		TargetEntityID: targetEntityID,
		Status:         dal.EntityMembershipStatusActive,
	}); err != nil {
		return nil, errors.Trace(err)
	}
	entity, err := s.dl.Entity(entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	pbEntity, err := getPBEntity(s.dl, entity, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), nil, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.CreateMembershipResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest) (out *directory.LookupEntitiesByContactResponse, err error) {
	entityContacts, err := s.dl.EntityContactsForValue(strings.TrimSpace(in.ContactValue))
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(entityContacts) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "Contact with value %s not found", in.ContactValue)
	}
	uniqueEntityIDs := make(map[uint64]struct{})
	var entityIDs []dal.EntityID
	for _, ec := range entityContacts {
		if _, ok := uniqueEntityIDs[ec.EntityID.ObjectID.Val]; !ok {
			entityIDs = append(entityIDs, ec.EntityID)
		}
	}
	statuses, err := transformEntityStatuses(in.Statuses)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	rootTypes, err := transformEntityTypes(in.RootTypes)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	childTypes, err := transformEntityTypes(in.ChildTypes)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	entities, err := s.dl.Entities(entityIDs, statuses, rootTypes)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if in.MemberOfEntity != "" {
		// ensure that memberships is in the requested information
		if in.RequestedInformation == nil {
			in.RequestedInformation = &directory.RequestedInformation{}
		}

		in.RequestedInformation.EntityInformation = appendMembershipEntityInformation(in.RequestedInformation.EntityInformation)
	}

	pbEntities, err := getPBEntities(s.dl, entities, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), statuses, childTypes)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.LookupEntitiesByContactResponse{
		Entities: filterToMembersOnly(pbEntities, in.MemberOfEntity),
	}, nil
}

func (s *server) LookupEntityDomain(ctx context.Context, in *directory.LookupEntityDomainRequest) (*directory.LookupEntityDomainResponse, error) {
	var err error
	var entityID *dal.EntityID
	if in.EntityID != "" {
		eID, err := dal.ParseEntityID(in.EntityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		entityID = &eID
	}

	queriedEntityID, queriedDomain, err := s.dl.EntityDomain(entityID, ptr.String(in.Domain))
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "entity_domain not found")
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &directory.LookupEntityDomainResponse{
		Domain:   queriedDomain,
		EntityID: queriedEntityID.String(),
	}, nil
}

func (s *server) CreateEntityDomain(ctx context.Context, in *directory.CreateEntityDomainRequest) (*directory.CreateEntityDomainResponse, error) {
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "entity_id required")
	} else if in.Domain == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "domain required")
	} else if len(in.Domain) > 255 {
		return nil, grpc.Errorf(codes.InvalidArgument, "domain can only be 255 characters in length")
	}

	eID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err := s.dl.UpsertEntityDomain(eID, strings.ToLower(in.Domain)); err != nil {
		return nil, errors.Trace(err)
	}

	return &directory.CreateEntityDomainResponse{}, nil
}

func (s *server) UpdateEntityDomain(ctx context.Context, in *directory.UpdateEntityDomainRequest) (*directory.UpdateEntityDomainResponse, error) {
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "entity_id required")
	} else if in.Domain == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "domain required")
	} else if len(in.Domain) > 255 {
		return nil, grpc.Errorf(codes.InvalidArgument, "domain can only be 255 characters in length")
	}

	eiD, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err := s.dl.Transact(func(dl dal.DAL) error {
		// ensure that a domain for the entity exists before updating it
		if _, _, err := dl.EntityDomain(&eiD, nil, dal.ForUpdate); errors.Cause(err) == dal.ErrNotFound {
			return errors.Trace(fmt.Errorf("directory: cannot update domain for an entity %s that does not have a domain", eiD.String()))
		}

		// ensure that no one else already has the domain being requested
		if entityID, _, err := dl.EntityDomain(nil, ptr.String(in.Domain)); err != nil && errors.Cause(err) != dal.ErrNotFound {
			return errors.Trace(err)
		} else if err == nil && entityID.String() != "" && entityID.String() != in.EntityID {
			return errors.Trace(fmt.Errorf("directory: domain %s already taken", in.Domain))
		}

		if err := dl.UpsertEntityDomain(eiD, strings.ToLower(in.Domain)); err != nil {
			return errors.Trace(err)
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &directory.UpdateEntityDomainResponse{}, nil
}

func (s *server) CreateContact(ctx context.Context, in *directory.CreateContactRequest) (out *directory.CreateContactResponse, err error) {
	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id")
	}
	if exists, err := doesEntityExist(s.dl, entityID); err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, grpc.Errorf(codes.NotFound, "Entity %s not found", in.EntityID)
	}
	if err := validateContact(in.GetContact()); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}

	contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(in.GetContact().ContactType)])
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unknown contact type: %v", in.GetContact().ContactType)
	}
	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		if _, err := dl.InsertEntityContact(&dal.EntityContact{
			EntityID:    entityID,
			Type:        contactType,
			Value:       in.GetContact().Value,
			Provisioned: in.GetContact().Provisioned,
			Label:       in.GetContact().Label,
			Verified:    in.GetContact().Verified,
		}); err != nil {
			return errors.Trace(err)
		}

		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), nil, nil)
		if err != nil {
			return errors.Trace(err)
		}

		if displayName := buildDisplayName(pbEntity.Info, pbEntity.Contacts); len(displayName) > 0 {
			pbEntity.Info.DisplayName = displayName
			if _, err := dl.UpdateEntity(entityID, &dal.EntityUpdate{
				DisplayName: ptr.String(displayName),
			}); err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.CreateContactResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) CreateContacts(ctx context.Context, in *directory.CreateContactsRequest) (out *directory.CreateContactsResponse, err error) {
	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id")
	}
	if exists, err := doesEntityExist(s.dl, entityID); err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, grpc.Errorf(codes.NotFound, "Entity %s not found", in.EntityID)
	}

	contacts := make([]*dal.EntityContact, len(in.Contacts))
	for i, c := range in.Contacts {
		if err := validateContact(c); err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
		}

		contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(c.ContactType)])
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "Unknown contact type: %v", c.ContactType)
		}
		contacts[i] = &dal.EntityContact{
			EntityID:    entityID,
			Type:        contactType,
			Value:       c.Value,
			Provisioned: c.Provisioned,
			Label:       c.Label,
			Verified:    c.Verified,
		}
	}

	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		if err := dl.InsertEntityContacts(contacts); err != nil {
			return errors.Trace(err)
		}
		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), nil, nil)
		if err != nil {
			return errors.Trace(err)
		}
		if displayName := buildDisplayName(pbEntity.Info, pbEntity.Contacts); len(displayName) > 0 {
			pbEntity.Info.DisplayName = displayName
			if _, err := dl.UpdateEntity(entityID, &dal.EntityUpdate{
				DisplayName: ptr.String(displayName),
			}); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.CreateContactsResponse{
		Entity: pbEntity,
	}, nil
}

func validateContact(contact *directory.Contact) error {
	switch contact.ContactType {
	case directory.ContactType_EMAIL:
		if !validate.Email(contact.Value) {
			return fmt.Errorf("Invalid email: %s", contact.Value)
		}
	case directory.ContactType_PHONE:
		if _, err := phone.ParseNumber(contact.Value); err != nil {
			return fmt.Errorf("Invalid phone number: %s", contact.Value)
		}
	}
	return nil
}

func (s *server) UpdateContacts(ctx context.Context, in *directory.UpdateContactsRequest) (out *directory.UpdateContactsResponse, err error) {
	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id: %s", in.EntityID)
	}
	if exists, err := doesEntityExist(s.dl, entityID); err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, grpc.Errorf(codes.NotFound, "Entity %s not found", in.EntityID)
	}

	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		for _, c := range in.Contacts {
			if c.ID == "" {
				return grpc.Errorf(codes.InvalidArgument, "A contact ID must be provided for all contacts being updated")
			}
			cID, err := dal.ParseEntityContactID(c.ID)
			if err != nil {
				return grpc.Errorf(codes.InvalidArgument, "Unable to parse contact id: %s", c.ID)
			}
			contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(c.ContactType)])
			if err != nil {
				return grpc.Errorf(codes.InvalidArgument, "Unknown contact type: %v", c.ContactType)
			}
			aff, err := s.dl.UpdateEntityContact(cID, &dal.EntityContactUpdate{
				Type:     &contactType,
				Value:    &c.Value,
				Label:    &c.Label,
				Verified: &c.Verified,
			})
			if err != nil {
				return errors.Trace(err)
			} else if aff == 0 {
				return grpc.Errorf(codes.NotFound, "Contact with ID %s was not found", c.ID)
			}
		}
		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), nil, nil)
		if err != nil {
			return errors.Trace(err)
		}

		if displayName := buildDisplayName(pbEntity.Info, pbEntity.Contacts); len(displayName) > 0 {
			pbEntity.Info.DisplayName = displayName
			if _, err := dl.UpdateEntity(entityID, &dal.EntityUpdate{
				DisplayName: ptr.String(displayName),
			}); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.UpdateContactsResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) DeleteContacts(ctx context.Context, in *directory.DeleteContactsRequest) (out *directory.DeleteContactsResponse, err error) {
	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id: %s", in.EntityID)
	}
	if exists, err := doesEntityExist(s.dl, entityID); err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, grpc.Errorf(codes.NotFound, "Entity %s not found", in.EntityID)
	}

	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		for _, cID := range in.EntityContactIDs {
			cID, err := dal.ParseEntityContactID(cID)
			if err != nil {
				return grpc.Errorf(codes.InvalidArgument, "Unable to parse contact id: %s", cID)
			}
			// TODO: Optimization here is to do a multinsert instead.
			_, err = s.dl.DeleteEntityContact(cID)
			if err != nil {
				return errors.Trace(err)
			}
		}
		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(in.RequestedInformation), riDepth(in.RequestedInformation), []dal.EntityStatus{}, nil)
		if err != nil {
			return errors.Trace(err)
		}

		if displayName := buildDisplayName(pbEntity.Info, pbEntity.Contacts); len(displayName) > 0 {
			pbEntity.Info.DisplayName = displayName
			if _, err := dl.UpdateEntity(entityID, &dal.EntityUpdate{
				DisplayName: ptr.String(displayName),
			}); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.DeleteContactsResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) DeleteEntity(ctx context.Context, in *directory.DeleteEntityRequest) (out *directory.DeleteEntityResponse, err error) {
	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Unable to parse entity id: %s", in.EntityID)
	}
	deleted := dal.EntityStatusDeleted
	if _, err := s.dl.UpdateEntity(entityID, &dal.EntityUpdate{
		Status: &deleted,
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &directory.DeleteEntityResponse{}, nil
}

func (s *server) Profile(ctx context.Context, in *directory.ProfileRequest) (*directory.ProfileResponse, error) {
	var profile *dal.EntityProfile
	switch in.LookupKeyType {
	case directory.ProfileRequest_ENTITY_ID:
		entID, err := dal.ParseEntityID(in.GetEntityID())
		if err != nil {
			return nil, errors.Trace(err)
		}
		profile, err = s.dl.EntityProfileForEntity(entID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpc.Errorf(codes.NotFound, "Profile for entity id %s not found", entID)
		} else if err != nil {
			return nil, errors.Trace(err)
		}
	case directory.ProfileRequest_PROFILE_ID:
		profileID, err := dal.ParseEntityProfileID(in.GetProfileID())
		if err != nil {
			return nil, errors.Trace(err)
		}
		profile, err = s.dl.EntityProfile(profileID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpc.Errorf(codes.NotFound, "Profile for profile id %s not found", profileID)
		} else if err != nil {
			return nil, errors.Trace(err)
		}
	default:
		return nil, errors.Errorf("Unknown lookup key type %s", in.LookupKeyType.String())
	}
	if profile == nil {
		return nil, errors.Errorf("No profile set after lookup for key type %s", in.LookupKeyType.String())
	}
	return &directory.ProfileResponse{
		Profile: transformEntityProfileToResponse(profile),
	}, nil
}

// UpdateProfile creates the profile if one does not exist
func (s *server) UpdateProfile(ctx context.Context, in *directory.UpdateProfileRequest) (*directory.UpdateProfileResponse, error) {
	var err error
	pID := dal.EmptyEntityProfileID()
	if in.Profile == nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "Profile required")
	}
	if in.ProfileID != "" {
		pID, err = dal.ParseEntityProfileID(in.ProfileID)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
		}
	} else {
		eID, err := dal.ParseEntityID(in.Profile.EntityID)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
		}
		entP, err := s.dl.EntityProfileForEntity(eID)
		if err != nil && errors.Cause(err) != dal.ErrNotFound {
			return nil, errors.Trace(err)
		}
		// If a profile already exists for this entity ID map it to the profile ID even if one wasn't supplied
		if entP != nil {
			pID = entP.ID
		}
	}

	// Assert the profile exists if one was supplied and map the entity
	if pID.IsValid {
		oldProfile, err := s.dl.EntityProfile(pID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpc.Errorf(codes.NotFound, "Profile id %s not found", pID)
		} else if err != nil {
			return nil, errors.Trace(err)
		}

		// Do not allow profiles to be remapped to different entities
		if in.Profile.EntityID != "" && in.Profile.EntityID != oldProfile.EntityID.String() {
			return nil, grpc.Errorf(codes.PermissionDenied, "The owning entity of a profile cannot be changed - Is: %s, Request Provided: %s", oldProfile.EntityID, in.Profile.EntityID)
		}
		in.Profile.EntityID = oldProfile.EntityID.String()
	}

	entID, err := dal.ParseEntityID(in.Profile.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := s.dl.Transact(func(dl dal.DAL) error {
		pID, err = dl.UpsertEntityProfile(&dal.EntityProfile{
			ID:       pID,
			EntityID: entID,
			Sections: &directory.ProfileSections{Sections: in.Profile.Sections},
		})
		if err != nil {
			return err
		}
		var imageMediaID *string
		if in.ImageMediaID != "" {
			imageMediaID = &in.ImageMediaID
		}

		_, err := dl.UpdateEntity(entID, &dal.EntityUpdate{
			CustomDisplayName: &in.Profile.DisplayName,
			FirstName:         &in.Profile.FirstName,
			LastName:          &in.Profile.LastName,
			HasProfile:        ptr.Bool(true),
			ImageMediaID:      imageMediaID,
		})
		return err
	}); err != nil {
		return nil, errors.Trace(err)
	}

	// Reread our profile to get any triggered modified times
	profileResp, err := s.Profile(ctx, &directory.ProfileRequest{
		LookupKeyType: directory.ProfileRequest_PROFILE_ID,
		LookupKeyOneof: &directory.ProfileRequest_ProfileID{
			ProfileID: pID.String(),
		},
	})
	if err != nil {
		// Trust the grpc error of the server call
		return nil, err
	}

	// Reread out entity to get any triggered modified times
	ent, err := s.dl.Entity(entID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	entResp, err := transformEntityToResponse(ent)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.UpdateProfileResponse{
		Profile: profileResp.Profile,
		Entity:  entResp,
	}, nil
}

func (s *server) CreateExternalLink(ctx context.Context, in *directory.CreateExternalLinkRequest) (*directory.CreateExternalLinkResponse, error) {
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "entity_id required")
	} else if in.Name == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "name required")
	} else if in.URL == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "url required")
	}

	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid entity_id %s : %s", in.EntityID, err)
	}

	if err := s.dl.InsertExternalLinkForEntity(entityID, in.Name, in.URL); err != nil {
		return nil, errors.Errorf("unable to insert external link (%s, %s) for %s : %s", in.Name, in.URL, entityID, err)
	}

	return &directory.CreateExternalLinkResponse{}, nil
}

func (s *server) DeleteExternalLink(ctx context.Context, in *directory.DeleteExternalLinkRequest) (*directory.DeleteExternalLinkResponse, error) {
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "entity_id required")
	} else if in.Name == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "name required")
	}

	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid entity_id %s : %s", in.EntityID, err)
	}

	if err := s.dl.DeleteExternalLinkForEntity(entityID, in.Name); err != nil {
		return nil, errors.Errorf("unable to delete external link (%s, %s) : %s", entityID, in.Name, err)
	}

	return &directory.DeleteExternalLinkResponse{}, nil
}

func (s *server) LookupExternalLinksForEntity(ctx context.Context, in *directory.LookupExternalLinksForEntityRequest) (*directory.LookupExternalLinksforEntityResponse, error) {
	if in.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "entity_id required")
	}

	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid entity_id %s : %s", in.EntityID, err)
	}

	externalLinks, err := s.dl.ExternalLinksForEntity(entityID)
	if err != nil {
		return nil, errors.Errorf("unable to get ehr links for %s : %s", entityID, err)
	}

	transformedExternalLinks := make([]*directory.LookupExternalLinksforEntityResponse_ExternalLink, len(externalLinks))
	for i, externalLink := range externalLinks {
		transformedExternalLinks[i] = &directory.LookupExternalLinksforEntityResponse_ExternalLink{
			Name: externalLink.Name,
			URL:  externalLink.URL,
		}
	}
	return &directory.LookupExternalLinksforEntityResponse{
		Links: transformedExternalLinks,
	}, nil
}

func (s *server) Contact(ctx context.Context, in *directory.ContactRequest) (*directory.ContactResponse, error) {
	if in.ContactID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "contact_id required")
	}
	contactID, err := dal.ParseEntityContactID(in.ContactID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid contact_id %s : %s", in.ContactID, err)
	}
	contact, err := s.dl.EntityContact(contactID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Contact %s NOT FOUND", in.ContactID)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.ContactResponse{
		Contact: transformEntityContactToResponse(contact),
	}, nil
}

func getPBEntities(dl dal.DAL, dEntities []*dal.Entity, entityInformation []directory.EntityInformation, depth int64, statuses []dal.EntityStatus, types []dal.EntityType) ([]*directory.Entity, error) {
	pbEntities := make([]*directory.Entity, len(dEntities))
	for i, e := range dEntities {
		pbEntity, err := getPBEntity(dl, e, entityInformation, depth, statuses, types)
		if err != nil {
			return nil, errors.Trace(err)
		}
		pbEntities[i] = pbEntity
	}
	return pbEntities, nil
}

// Note: How we optimize this deep crawl is very likely to change
func getPBEntity(dl dal.DAL, dEntity *dal.Entity, entityInformation []directory.EntityInformation, depth int64, statuses []dal.EntityStatus, types []dal.EntityType) (*directory.Entity, error) {
	id := dEntity.ID
	entity, err := transformEntityToResponse(dEntity)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if depth >= 0 {
		if hasRequestedInfo(entityInformation, directory.EntityInformation_MEMBERSHIPS) {
			memberships, err := getPBMemberships(dl, id, entityInformation, depth-1, statuses, types)
			if err != nil {
				return nil, errors.Trace(err)
			}
			entity.Memberships = memberships
		}
		if hasRequestedInfo(entityInformation, directory.EntityInformation_MEMBERS) {

			members, err := getPBMembers(dl, id, entityInformation, depth-1, statuses, types)
			if err != nil {
				return nil, errors.Trace(err)
			}
			entity.Members = members
		}
		if hasRequestedInfo(entityInformation, directory.EntityInformation_EXTERNAL_IDS) {
			externalIDs, err := getPBExternalIDs(dl, id)
			if err != nil {
				return nil, errors.Trace(err)
			}
			entity.ExternalIDs = externalIDs
		}
		if hasRequestedInfo(entityInformation, directory.EntityInformation_CONTACTS) {
			contacts, err := getPBEntityContacts(dl, id)
			if err != nil {
				return nil, errors.Trace(err)
			}
			entity.Contacts = contacts
		}
		entity.IncludedInformation = entityInformation
	}
	return entity, nil
}

func getPBMemberships(dl dal.DAL, entityID dal.EntityID, entityInformation []directory.EntityInformation, depth int64, statuses []dal.EntityStatus, types []dal.EntityType) ([]*directory.Entity, error) {
	memberships, err := dl.EntityMemberships(entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	entityIDs := make([]dal.EntityID, len(memberships))
	for i, membership := range memberships {
		entityIDs[i] = membership.TargetEntityID
	}
	entities, err := dl.Entities(entityIDs, statuses, types)
	if err != nil {
		return nil, errors.Trace(err)
	}
	pbEntities, err := getPBEntities(dl, entities, entityInformation, depth, statuses, types)
	return pbEntities, errors.Trace(err)
}

func getPBMembers(dl dal.DAL, entityID dal.EntityID, entityInformation []directory.EntityInformation, depth int64, statuses []dal.EntityStatus, types []dal.EntityType) ([]*directory.Entity, error) {
	members, err := dl.EntityMembers(entityID, statuses, types)
	if err != nil {
		return nil, errors.Trace(err)
	}
	pbEntities, err := getPBEntities(dl, members, entityInformation, depth, statuses, types)
	return pbEntities, errors.Trace(err)
}

func getPBExternalIDs(dl dal.DAL, entityID dal.EntityID) ([]string, error) {
	externalIDs, err := dl.ExternalEntityIDsForEntities([]dal.EntityID{entityID})
	if err != nil {
		return nil, errors.Trace(err)
	}
	externalIDStrings := make([]string, len(externalIDs))
	for i, eID := range externalIDs {
		externalIDStrings[i] = eID.ExternalID
	}
	return externalIDStrings, nil
}

func getPBEntityContacts(dl dal.DAL, entityID dal.EntityID) ([]*directory.Contact, error) {
	entityContacts, err := dl.EntityContacts(entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	pbEntityContacts := make([]*directory.Contact, len(entityContacts))
	for i, ec := range entityContacts {
		pbEntityContacts[i] = transformEntityContactToResponse(ec)
	}
	return pbEntityContacts, nil
}

// Note: Assuming this is small it's not a big deal but might want to be more efficient if it grows
func hasRequestedInfo(requestedInformation []directory.EntityInformation, target directory.EntityInformation) bool {
	for _, v := range requestedInformation {
		if v == target {
			return true
		}
	}
	return false
}

func buildDisplayName(info *directory.EntityInfo, contacts []*directory.Contact) string {
	if info.FirstName != "" || info.LastName != "" {
		var displayName string
		if info.FirstName != "" {
			displayName = info.FirstName
		}
		if info.MiddleInitial != "" {
			displayName += " " + info.MiddleInitial
		}
		if info.LastName != "" {
			displayName += " " + info.LastName
		}

		if info.ShortTitle != "" {
			displayName += ", " + info.ShortTitle
		}
		return displayName
	} else if info.GroupName != "" {
		return info.GroupName
	}

	// pick the display name to be the first contact value
	for _, c := range contacts {
		if c.ContactType == directory.ContactType_PHONE {
			pn, err := phone.Format(c.Value, phone.Pretty)
			if err != nil {
				return c.Value
			}
			return pn
		}
		return c.Value
	}

	return ""
}
