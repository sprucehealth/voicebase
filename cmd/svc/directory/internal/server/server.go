package server

import (
	"fmt"
	"strings"

	"context"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrf = grpc.Errorf

func grpcErrorf(c codes.Code, format string, a ...interface{}) error {
	if c == codes.Internal {
		golog.LogDepthf(1, golog.ERR, "Directory - Internal GRPC Error: %s", fmt.Sprintf(format, a...))
	}
	return grpcErrf(c, format, a...)
}

func grpcError(err error) error {
	if grpc.Code(err) == codes.Unknown {
		return grpcErrorf(codes.Internal, err.Error())
	}
	return err
}

var (
	// ErrNotImplemented is returned from RPC calls that have yet to be implemented
	ErrNotImplemented = errors.New("Not Implemented")
)

type server struct {
	dl                         dal.DAL
	statLookupEntitiesEntities *metrics.Counter
}

// New returns an initialized instance of server
func New(dl dal.DAL, metricsRegistry metrics.Registry) directory.DirectoryServer {
	srv := &server{
		dl: dl,
		statLookupEntitiesEntities: metrics.NewCounter(),
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

	var membershipFound bool
	for _, ei := range entityInfo {
		if ei == directory.EntityInformation_MEMBERSHIPS {
			membershipFound = true
			break
		}
	}

	if !membershipFound {
		entityInfo = append(entityInfo, directory.EntityInformation_MEMBERSHIPS)
	}

	return entityInfo
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

func (s *server) LookupEntities(ctx context.Context, rd *directory.LookupEntitiesRequest) (out *directory.LookupEntitiesResponse, err error) {
	var entityIDs []dal.EntityID
	switch rd.LookupKeyType {
	case directory.LookupEntitiesRequest_EXTERNAL_ID, directory.LookupEntitiesRequest_ACCOUNT_ID:
		var extID string
		switch rd.LookupKeyType {
		case directory.LookupEntitiesRequest_EXTERNAL_ID:
			extID = rd.GetExternalID()
		// TODO: Actually use the account_id field on the table and don't do this double lookup
		case directory.LookupEntitiesRequest_ACCOUNT_ID:
			extID = rd.GetAccountID()
		}
		externalEntityIDs, err := s.dl.ExternalEntityIDs(extID)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
		entityIDs = make([]dal.EntityID, len(externalEntityIDs))
		for i, v := range externalEntityIDs {
			entityIDs[i] = v.EntityID
		}
	case directory.LookupEntitiesRequest_ENTITY_ID:
		entityID, err := dal.ParseEntityID(rd.GetEntityID())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id")
		}
		entityIDs = append(entityIDs, entityID)
	case directory.LookupEntitiesRequest_BATCH_ENTITY_ID:
		idList := rd.GetBatchEntityID().IDs
		entityIDs = make([]dal.EntityID, len(idList))
		for i, id := range idList {
			eid, err := dal.ParseEntityID(id)
			if err != nil {
				return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id")
			}
			entityIDs[i] = eid
		}
	default:
		return nil, grpcErrorf(codes.Internal, "Unknown lookup key type %s", rd.LookupKeyType.String())
	}
	statuses, err := transformEntityStatuses(rd.Statuses)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}

	rootTypes, err := transformEntityTypes(rd.RootTypes)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}

	childTypes, err := transformEntityTypes(rd.ChildTypes)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}

	entities, err := s.dl.Entities(entityIDs, statuses, rootTypes)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	if len(entities) == 0 {
		return nil, grpcErrorf(codes.NotFound, "No entities located matching query")
	}

	if rd.MemberOfEntity != "" {
		// ensure that memberships is in the requested information
		if rd.RequestedInformation == nil {
			rd.RequestedInformation = &directory.RequestedInformation{}
		}

		rd.RequestedInformation.EntityInformation = appendMembershipEntityInformation(rd.RequestedInformation.EntityInformation)
	}

	pbEntities, err := getPBEntities(s.dl, entities, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), statuses, childTypes)
	if err != nil {
		return nil, errors.Trace(err)
	}

	s.statLookupEntitiesEntities.Inc(uint64(len(pbEntities)))
	return &directory.LookupEntitiesResponse{
		Entities: filterToMembersOnly(pbEntities, rd.MemberOfEntity),
	}, nil
}

func (s *server) CreateEntity(ctx context.Context, rd *directory.CreateEntityRequest) (out *directory.CreateEntityResponse, err error) {
	if err := s.validateCreateEntityRequest(rd); err != nil {
		return nil, err
	}

	entityType, err := dal.ParseEntityType(directory.EntityType_name[int32(rd.Type)])
	if err != nil {
		return nil, errors.Trace(err)
	}
	var pbEntity *directory.Entity
	if err = s.dl.Transact(func(dl dal.DAL) error {
		var entityGender *dal.EntityGender
		if rd.EntityInfo.Gender != directory.EntityInfo_UNKNOWN {
			eg, err := dal.ParseEntityGender(rd.EntityInfo.Gender.String())
			if err != nil {
				return errors.Trace(err)
			}
			entityGender = &eg
		}
		var dob *encoding.Date
		if rd.EntityInfo.DOB != nil {
			dob = &encoding.Date{
				Month: int(rd.EntityInfo.DOB.Month),
				Day:   int(rd.EntityInfo.DOB.Day),
				Year:  int(rd.EntityInfo.DOB.Year),
			}
		}

		var displayName string
		if rd.EntityInfo.DisplayName != "" {
			displayName = rd.EntityInfo.DisplayName
		} else {
			displayName = buildDisplayName(rd.EntityInfo, rd.Contacts)
			if len(displayName) == 0 {
				return errors.Trace(errors.New("Not enough information to build the display name for entity"))
			}
		}

		entityID, err := dl.InsertEntity(&dal.Entity{
			Type:          entityType,
			Status:        dal.EntityStatusActive,
			FirstName:     rd.EntityInfo.FirstName,
			MiddleInitial: rd.EntityInfo.MiddleInitial,
			LastName:      rd.EntityInfo.LastName,
			GroupName:     rd.EntityInfo.GroupName,
			DisplayName:   displayName,
			ShortTitle:    rd.EntityInfo.ShortTitle,
			LongTitle:     rd.EntityInfo.LongTitle,
			Gender:        entityGender,
			DOB:           dob,
			AccountID:     rd.AccountID,
			Note:          rd.EntityInfo.Note,
		})
		if err != nil {
			return errors.Trace(err)
		}
		if rd.ExternalID != "" {
			if err := dl.InsertExternalEntityID(&dal.ExternalEntityID{
				EntityID:   entityID,
				ExternalID: rd.ExternalID,
			}); err != nil {
				return errors.Trace(err)
			}
		}
		if rd.InitialMembershipEntityID != "" {
			targetEntityID, err := dal.ParseEntityID(rd.InitialMembershipEntityID)
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
		for _, contact := range rd.Contacts {
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
			}); err != nil {
				return errors.Trace(err)
			}
		}
		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil, nil)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.CreateEntityResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) CreateExternalIDs(ctx context.Context, rd *directory.CreateExternalIDsRequest) (out *directory.CreateExternalIDsResponse, err error) {
	if rd.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "EntityID cannot be empty")
	}
	if err = s.dl.Transact(func(dl dal.DAL) error {
		models := make([]*dal.ExternalEntityID, 0, len(rd.ExternalIDs))
		for _, eID := range rd.ExternalIDs {
			entityID, err := dal.ParseEntityID(rd.EntityID)
			if err != nil {
				return grpcErrorf(codes.InvalidArgument, err.Error())
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

func (s *server) SerializedEntityContact(ctx context.Context, rd *directory.SerializedEntityContactRequest) (out *directory.SerializedEntityContactResponse, err error) {
	if rd.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity id required")
	}
	entityID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Error parsing entity id: %s", err)
	}
	platform, err := dal.ParseSerializedClientEntityContactPlatform(directory.Platform_name[int32(rd.Platform)])
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Error parsing platform type: %s", err)
	}
	sec, err := s.dl.SerializedClientEntityContact(entityID, platform)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "No serialized entity contact exists for entity id %s and platform %s", entityID, platform)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &directory.SerializedEntityContactResponse{
		SerializedEntityContact: transformSerializedClientEntityContactToResponse(sec),
	}, nil
}

func (s *server) validateCreateEntityRequest(rd *directory.CreateEntityRequest) error {
	if rd.InitialMembershipEntityID != "" {
		eID, err := dal.ParseEntityID(rd.InitialMembershipEntityID)
		if err != nil {
			return grpcErrorf(codes.InvalidArgument, "Unable to parse entity id")
		}
		exists, err := doesEntityExist(s.dl, eID)
		if err != nil {
			return grpcErrorf(codes.Internal, err.Error())
		}
		if !exists {
			return grpcErrorf(codes.NotFound, "Entity not found %s", rd.InitialMembershipEntityID)
		}
	}
	if rd.EntityInfo == nil {
		rd.EntityInfo = &directory.EntityInfo{}
	}
	for _, contact := range rd.Contacts {
		if contact.Value == "" {
			return grpcErrorf(codes.InvalidArgument, "Contact value cannot be empty")
		}
		if err := validateContact(contact); err != nil {
			return grpcErrorf(codes.InvalidArgument, err.Error())
		}
	}
	return nil
}

func (s *server) UpdateEntity(ctx context.Context, rd *directory.UpdateEntityRequest) (out *directory.UpdateEntityResponse, err error) {
	oldEnt, err := s.validateUpdateEntityRequest(rd)
	if err != nil {
		return nil, err
	}
	defer oldEnt.Recycle()

	eID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity ID")
	}

	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		entityUpdate := &dal.EntityUpdate{}

		entityInfo := transformEntityToEntityInfoResponse(oldEnt)
		if rd.UpdateEntityInfo {
			entityInfo = rd.EntityInfo
			entityUpdate.FirstName = &rd.EntityInfo.FirstName
			entityUpdate.MiddleInitial = &rd.EntityInfo.MiddleInitial
			entityUpdate.LastName = &rd.EntityInfo.LastName
			entityUpdate.GroupName = &rd.EntityInfo.GroupName
			entityUpdate.ShortTitle = &rd.EntityInfo.ShortTitle
			entityUpdate.LongTitle = &rd.EntityInfo.LongTitle
			entityUpdate.Note = &rd.EntityInfo.Note

			if rd.EntityInfo.Gender != directory.EntityInfo_UNKNOWN {
				g, err := dal.ParseEntityGender(rd.EntityInfo.Gender.String())
				if err != nil {
					return grpcErrorf(codes.InvalidArgument, "Unknown entity gender %s", rd.EntityInfo.Gender.String())
				}
				entityUpdate.Gender = &g
			}
			if rd.EntityInfo.DOB != nil {
				entityUpdate.DOB = &encoding.Date{
					Month: int(rd.EntityInfo.DOB.Month),
					Day:   int(rd.EntityInfo.DOB.Day),
					Year:  int(rd.EntityInfo.DOB.Year),
				}
			}
		}

		if rd.UpdateImageMediaID {
			entityUpdate.ImageMediaID = &rd.ImageMediaID
		}

		if rd.UpdateAccountID {
			entityUpdate.AccountID = &rd.AccountID
		}

		var contacts []*directory.Contact
		if rd.UpdateContacts {
			// Delete existing contact info
			if _, err := dl.DeleteEntityContactsForEntityID(eID); err != nil {
				return errors.Trace(err)
			}

			// Insert the new set of contacts
			if len(rd.Contacts) != 0 {
				dalContacts := make([]*dal.EntityContact, len(rd.Contacts))
				for i, contact := range rd.Contacts {
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
					}
				}
				if err := dl.InsertEntityContacts(dalContacts); err != nil {
					return errors.Trace(err)
				}

				contacts = rd.Contacts
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
		if rd.UpdateSerializedEntityContacts {
			for _, sec := range rd.SerializedEntityContacts {
				platform, err := dal.ParseSerializedClientEntityContactPlatform(directory.Platform_name[int32(sec.Platform)])
				if err != nil {
					return grpcErrorf(codes.InvalidArgument, "Error parsing platform type: %s", err)
				}
				if err := dl.UpsertSerializedClientEntityContact(&dal.SerializedClientEntityContact{
					EntityID:                eID,
					Platform:                platform,
					SerializedEntityContact: sec.SerializedEntityContact,
				}); err != nil {
					return grpcErrorf(codes.Internal, err.Error())
				}
			}
		}

		entity, err := dl.Entity(eID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil, nil)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.UpdateEntityResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) validateUpdateEntityRequest(rd *directory.UpdateEntityRequest) (*dal.Entity, error) {
	eID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity ID")
	}
	ent, err := s.dl.Entity(eID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, err.Error())
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return ent, nil
}

func (s *server) ExternalIDs(ctx context.Context, rd *directory.ExternalIDsRequest) (out *directory.ExternalIDsResponse, err error) {
	ids := make([]dal.EntityID, len(rd.EntityIDs))
	for i, id := range rd.EntityIDs {
		eID, err := dal.ParseEntityID(id)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id")
		}
		ids[i] = eID
	}
	externalIDs, err := s.dl.ExternalEntityIDsForEntities(ids)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
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

func (s *server) CreateMembership(ctx context.Context, rd *directory.CreateMembershipRequest) (*directory.CreateMembershipResponse, error) {
	entityID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id")
	}
	targetEntityID, err := dal.ParseEntityID(rd.TargetEntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id")
	}
	exists, err := doesEntityExist(s.dl, entityID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	if !exists {
		return nil, grpcErrorf(codes.NotFound, "Entity not found %s", rd.EntityID)
	}
	exists, err = doesEntityExist(s.dl, targetEntityID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	if !exists {
		return nil, grpcErrorf(codes.NotFound, "Entity not found %s", rd.TargetEntityID)
	}

	if err := s.dl.InsertEntityMembership(&dal.EntityMembership{
		EntityID:       entityID,
		TargetEntityID: targetEntityID,
		Status:         dal.EntityMembershipStatusActive,
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	entity, err := s.dl.Entity(entityID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	pbEntity, err := getPBEntity(s.dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil, nil)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &directory.CreateMembershipResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) LookupEntitiesByContact(ctx context.Context, rd *directory.LookupEntitiesByContactRequest) (out *directory.LookupEntitiesByContactResponse, err error) {
	entityContacts, err := s.dl.EntityContactsForValue(strings.TrimSpace(rd.ContactValue))
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	if len(entityContacts) == 0 {
		return nil, grpcErrorf(codes.NotFound, "Contact with value %s not found", rd.ContactValue)
	}
	uniqueEntityIDs := make(map[uint64]struct{})
	var entityIDs []dal.EntityID
	for _, ec := range entityContacts {
		if _, ok := uniqueEntityIDs[ec.EntityID.ObjectID.Val]; !ok {
			entityIDs = append(entityIDs, ec.EntityID)
		}
	}
	statuses, err := transformEntityStatuses(rd.Statuses)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	rootTypes, err := transformEntityTypes(rd.RootTypes)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	childTypes, err := transformEntityTypes(rd.ChildTypes)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	entities, err := s.dl.Entities(entityIDs, statuses, rootTypes)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	if rd.MemberOfEntity != "" {
		// ensure that memberships is in the requested information
		if rd.RequestedInformation == nil {
			rd.RequestedInformation = &directory.RequestedInformation{}
		}

		rd.RequestedInformation.EntityInformation = appendMembershipEntityInformation(rd.RequestedInformation.EntityInformation)
	}

	pbEntities, err := getPBEntities(s.dl, entities, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), statuses, childTypes)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &directory.LookupEntitiesByContactResponse{
		Entities: filterToMembersOnly(pbEntities, rd.MemberOfEntity),
	}, nil
}

func (s *server) LookupEntityDomain(ctx context.Context, in *directory.LookupEntityDomainRequest) (*directory.LookupEntityDomainResponse, error) {
	var err error
	var entityID *dal.EntityID
	if in.EntityID != "" {
		eID, err := dal.ParseEntityID(in.EntityID)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
		entityID = &eID
	}

	queriedEntityID, queriedDomain, err := s.dl.EntityDomain(entityID, ptr.String(in.Domain))
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpcErrorf(codes.NotFound, "entity_domain not found")
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &directory.LookupEntityDomainResponse{
		Domain:   queriedDomain,
		EntityID: queriedEntityID.String(),
	}, nil
}

func (s *server) CreateEntityDomain(ctx context.Context, in *directory.CreateEntityDomainRequest) (*directory.CreateEntityDomainResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity_id required")
	} else if in.Domain == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "domain required")
	} else if len(in.Domain) > 255 {
		return nil, grpcErrorf(codes.InvalidArgument, "domain can only be 255 characters in length")
	}

	eID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	if err := s.dl.UpsertEntityDomain(eID, strings.ToLower(in.Domain)); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &directory.CreateEntityDomainResponse{}, nil
}

func (s *server) UpdateEntityDomain(ctx context.Context, in *directory.UpdateEntityDomainRequest) (*directory.UpdateEntityDomainResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity_id required")
	} else if in.Domain == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "domain required")
	} else if len(in.Domain) > 255 {
		return nil, grpcErrorf(codes.InvalidArgument, "domain can only be 255 characters in length")
	}

	eiD, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
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
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &directory.UpdateEntityDomainResponse{}, nil
}

func (s *server) CreateContact(ctx context.Context, rd *directory.CreateContactRequest) (out *directory.CreateContactResponse, err error) {
	entityID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id")
	}
	if exists, err := doesEntityExist(s.dl, entityID); err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, grpcErrorf(codes.NotFound, "Entity %s not found", rd.EntityID)
	}
	if err := validateContact(rd.GetContact()); err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}

	contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(rd.GetContact().ContactType)])
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unknown contact type: %v", rd.GetContact().ContactType)
	}
	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		if _, err := dl.InsertEntityContact(&dal.EntityContact{
			EntityID:    entityID,
			Type:        contactType,
			Value:       rd.GetContact().Value,
			Provisioned: rd.GetContact().Provisioned,
			Label:       rd.GetContact().Label,
			Verified:    rd.GetContact().Verified,
		}); err != nil {
			return errors.Trace(err)
		}

		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil, nil)
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

func (s *server) CreateContacts(ctx context.Context, rd *directory.CreateContactsRequest) (out *directory.CreateContactsResponse, err error) {
	entityID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id")
	}
	if exists, err := doesEntityExist(s.dl, entityID); err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, grpcErrorf(codes.NotFound, "Entity %s not found", rd.EntityID)
	}

	contacts := make([]*dal.EntityContact, len(rd.Contacts))
	for i, c := range rd.Contacts {
		if err := validateContact(c); err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, err.Error())
		}

		contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(c.ContactType)])
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Unknown contact type: %v", c.ContactType)
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

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil, nil)
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

func (s *server) UpdateContacts(ctx context.Context, rd *directory.UpdateContactsRequest) (out *directory.UpdateContactsResponse, err error) {
	entityID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id: %s", rd.EntityID)
	}
	if exists, err := doesEntityExist(s.dl, entityID); err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, grpcErrorf(codes.NotFound, "Entity %s not found", rd.EntityID)
	}

	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		for _, c := range rd.Contacts {
			if c.ID == "" {
				return grpcErrorf(codes.InvalidArgument, "A contact ID must be provided for all contacts being updated")
			}
			cID, err := dal.ParseEntityContactID(c.ID)
			if err != nil {
				return grpcErrorf(codes.InvalidArgument, "Unable to parse contact id: %s", c.ID)
			}
			contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(c.ContactType)])
			if err != nil {
				return grpcErrorf(codes.InvalidArgument, "Unknown contact type: %v", c.ContactType)
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
				return grpcErrorf(codes.NotFound, "Contact with ID %s was not found", c.ID)
			}
		}
		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil, nil)
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

func (s *server) DeleteContacts(ctx context.Context, rd *directory.DeleteContactsRequest) (out *directory.DeleteContactsResponse, err error) {
	entityID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id: %s", rd.EntityID)
	}
	if exists, err := doesEntityExist(s.dl, entityID); err != nil {
		return nil, errors.Trace(err)
	} else if !exists {
		return nil, grpcErrorf(codes.NotFound, "Entity %s not found", rd.EntityID)
	}

	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		for _, cID := range rd.EntityContactIDs {
			cID, err := dal.ParseEntityContactID(cID)
			if err != nil {
				return grpcErrorf(codes.InvalidArgument, "Unable to parse contact id: %s", cID)
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

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), []dal.EntityStatus{}, nil)
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

func (s *server) DeleteEntity(ctx context.Context, rd *directory.DeleteEntityRequest) (out *directory.DeleteEntityResponse, err error) {
	entityID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity id: %s", rd.EntityID)
	}
	deleted := dal.EntityStatusDeleted
	if _, err := s.dl.UpdateEntity(entityID, &dal.EntityUpdate{
		Status: &deleted,
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &directory.DeleteEntityResponse{}, nil
}

func (s *server) Profile(ctx context.Context, rd *directory.ProfileRequest) (*directory.ProfileResponse, error) {
	var profile *dal.EntityProfile
	switch rd.LookupKeyType {
	case directory.ProfileRequest_ENTITY_ID:
		entID, err := dal.ParseEntityID(rd.GetEntityID())
		if err != nil {
			return nil, grpcError(err)
		}
		profile, err = s.dl.EntityProfileForEntity(entID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Profile for entity id %s not found", entID)
		} else if err != nil {
			return nil, grpcError(err)
		}
	case directory.ProfileRequest_PROFILE_ID:
		profileID, err := dal.ParseEntityProfileID(rd.GetProfileID())
		if err != nil {
			return nil, grpcError(err)
		}
		profile, err = s.dl.EntityProfile(profileID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Profile for profile id %s not found", profileID)
		} else if err != nil {
			return nil, grpcError(err)
		}
	default:
		return nil, grpcErrorf(codes.Internal, "Unknown lookup key type %s", rd.LookupKeyType.String())
	}
	if profile == nil {
		return nil, grpcError(fmt.Errorf("No profile set after lookup for key type %s", rd.LookupKeyType.String()))
	}
	return &directory.ProfileResponse{
		Profile: transformEntityProfileToResponse(profile),
	}, nil
}

// UpdateProfile creates the profile if one does not exist
func (s *server) UpdateProfile(ctx context.Context, rd *directory.UpdateProfileRequest) (*directory.UpdateProfileResponse, error) {
	var err error
	pID := dal.EmptyEntityProfileID()
	if rd.Profile == nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Profile required")
	}
	if rd.ProfileID != "" {
		pID, err = dal.ParseEntityProfileID(rd.ProfileID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, err.Error())
		}
	} else {
		eID, err := dal.ParseEntityID(rd.Profile.EntityID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, err.Error())
		}
		entP, err := s.dl.EntityProfileForEntity(eID)
		if err != nil && errors.Cause(err) != dal.ErrNotFound {
			return nil, grpcError(err)
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
			return nil, grpcErrorf(codes.NotFound, "Profile id %s not found", pID)
		} else if err != nil {
			return nil, grpcError(err)
		}

		// Do not allow profiles to be remapped to different entities
		if rd.Profile.EntityID != "" && rd.Profile.EntityID != oldProfile.EntityID.String() {
			return nil, grpcErrorf(codes.PermissionDenied, "The owning entity of a profile cannot be changed - Is: %s, Request Provided: %s", oldProfile.EntityID, rd.Profile.EntityID)
		}
		rd.Profile.EntityID = oldProfile.EntityID.String()
	}

	entID, err := dal.ParseEntityID(rd.Profile.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	if err := s.dl.Transact(func(dl dal.DAL) error {
		pID, err = dl.UpsertEntityProfile(&dal.EntityProfile{
			ID:       pID,
			EntityID: entID,
			Sections: &directory.ProfileSections{Sections: rd.Profile.Sections},
		})
		if err != nil {
			return err
		}
		var imageMediaID *string
		if rd.ImageMediaID != "" {
			imageMediaID = &rd.ImageMediaID
		}

		_, err := dl.UpdateEntity(entID, &dal.EntityUpdate{
			CustomDisplayName: &rd.Profile.DisplayName,
			FirstName:         &rd.Profile.FirstName,
			LastName:          &rd.Profile.LastName,
			HasProfile:        ptr.Bool(true),
			ImageMediaID:      imageMediaID,
		})
		return err
	}); err != nil {
		return nil, grpcError(err)
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
		return nil, grpcError(err)
	}
	entResp, err := transformEntityToResponse(ent)
	if err != nil {
		return nil, grpcError(err)
	}
	return &directory.UpdateProfileResponse{
		Profile: profileResp.Profile,
		Entity:  entResp,
	}, nil
}

func (s *server) CreateEHRLink(ctx context.Context, in *directory.CreateEHRLinkRequest) (*directory.CreateEHRLinkResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity_id required")
	} else if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name required")
	} else if in.URL == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "url required")
	}

	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "invalid entity_id %s : %s", in.EntityID, err)
	}

	if err := s.dl.InsertEHRLinkForEntity(entityID, in.Name, in.URL); err != nil {
		return nil, grpcErrorf(codes.Internal, "unable to insert ehr link (%s, %s) for %s : %s", in.Name, in.URL, entityID, err)
	}

	return &directory.CreateEHRLinkResponse{}, nil
}

func (s *server) DeleteEHRLink(ctx context.Context, in *directory.DeleteEHRLinkRequest) (*directory.DeleteEHRLinkResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity_id required")
	} else if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name required")
	}

	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "invalid entity_id %s : %s", in.EntityID, err)
	}

	if err := s.dl.DeleteEHRLinkForEntity(entityID, in.Name); err != nil {
		return nil, grpcErrorf(codes.Internal, "unable to delete ehr link (%s, %s) : %s", entityID, in.Name, err)
	}

	return &directory.DeleteEHRLinkResponse{}, nil
}

func (s *server) LookupEHRLinksForEntity(ctx context.Context, in *directory.LookupEHRLinksForEntityRequest) (*directory.LookupEHRLinksforEntityResponse, error) {
	if in.EntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "entity_id required")
	}

	entityID, err := dal.ParseEntityID(in.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "invalid entity_id %s : %s", in.EntityID, err)
	}

	ehrLinks, err := s.dl.EHRLinksForEntity(entityID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, "unable to get ehr links for %s : %s", entityID, err)
	}

	transformedEHRLinks := make([]*directory.LookupEHRLinksforEntityResponse_EHRLink, len(ehrLinks))
	for i, ehrLink := range ehrLinks {
		transformedEHRLinks[i] = &directory.LookupEHRLinksforEntityResponse_EHRLink{
			Name: ehrLink.Name,
			URL:  ehrLink.URL,
		}
	}
	return &directory.LookupEHRLinksforEntityResponse{
		Links: transformedEHRLinks,
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
