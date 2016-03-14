package server

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrorf = grpc.Errorf

var (
	// ErrNotImplemented is returned from RPC calls that have yet to be implemented
	ErrNotImplemented = errors.New("Not Implemented")
)

type server struct {
	dl dal.DAL
}

// New returns an initialized instance of server
func New(dl dal.DAL) directory.DirectoryServer {
	return &server{dl: dl}
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

func (s *server) LookupEntities(ctx context.Context, rd *directory.LookupEntitiesRequest) (out *directory.LookupEntitiesResponse, err error) {
	golog.Debugf("Entering server.server.LookupEntities: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.LookupEntities... %+v", out) }()
	}
	var entityIDs []dal.EntityID
	switch rd.LookupKeyType {
	case directory.LookupEntitiesRequest_EXTERNAL_ID:
		externalEntityIDs, err := s.dl.ExternalEntityIDs(rd.GetExternalID())
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
	default:
		return nil, grpcErrorf(codes.Internal, "Unknown lookup key type %d", rd.LookupKeyType)
	}
	statuses, err := transformEntityStatuses(rd.Statuses)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	entities, err := s.dl.Entities(entityIDs, statuses)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	if len(entities) == 0 {
		return nil, grpcErrorf(codes.NotFound, "No entities located matching query")
	}
	pbEntities, err := getPBEntities(s.dl, entities, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), statuses)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.LookupEntitiesResponse{
		Entities: pbEntities,
	}, nil
}

func (s *server) CreateEntity(ctx context.Context, rd *directory.CreateEntityRequest) (out *directory.CreateEntityResponse, err error) {
	golog.Debugf("Entering server.server.CreateEntity: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CreateEntity... %+v", out) }()
	}
	if err := s.validateCreateEntityRequest(rd); err != nil {
		return nil, err
	}

	entityType, err := dal.ParseEntityType(directory.EntityType_name[int32(rd.Type)])
	if err != nil {
		return nil, errors.Trace(err)
	}
	var pbEntity *directory.Entity
	if err = s.dl.Transact(func(dl dal.DAL) error {
		entityID, err := dl.InsertEntity(&dal.Entity{
			Type:          entityType,
			Status:        dal.EntityStatusActive,
			FirstName:     rd.EntityInfo.FirstName,
			MiddleInitial: rd.EntityInfo.MiddleInitial,
			LastName:      rd.EntityInfo.LastName,
			GroupName:     rd.EntityInfo.GroupName,
			DisplayName:   rd.EntityInfo.DisplayName,
			ShortTitle:    rd.EntityInfo.ShortTitle,
			LongTitle:     rd.EntityInfo.LongTitle,
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

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.CreateEntityResponse{
		Entity: pbEntity,
	}, nil
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
	if !api.IsErrNotFound(err) && err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if api.IsErrNotFound(err) {
		return nil, grpcErrorf(codes.NotFound, "No serialized entity contact exists for entity id %s and platform %s", entityID, platform)
	}
	return &directory.SerializedEntityContactResponse{
		SerializedEntityContact: dalSerializedClientEntityContactAsPBSerializedClientEntityContact(sec),
	}, nil
}

func (s *server) validateCreateEntityRequest(rd *directory.CreateEntityRequest) error {
	if rd.Type != directory.EntityType_EXTERNAL && rd.EntityInfo.DisplayName == "" {
		return grpcErrorf(codes.InvalidArgument, "DisplayName cannot be empty for non external entities")
	}
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
	golog.Debugf("Entering server.server.UpdateEntity: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.UpdateEntity... %+v", out) }()
	}
	if err := s.validateUpdateEntityRequest(rd); err != nil {
		return nil, err
	}

	eID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse entity ID")
	}

	var pbEntity *directory.Entity
	if err := s.dl.Transact(func(dl dal.DAL) error {
		_, err := dl.UpdateEntity(eID, &dal.EntityUpdate{
			FirstName:     &rd.EntityInfo.FirstName,
			MiddleInitial: &rd.EntityInfo.MiddleInitial,
			LastName:      &rd.EntityInfo.LastName,
			GroupName:     &rd.EntityInfo.GroupName,
			DisplayName:   &rd.EntityInfo.DisplayName,
			ShortTitle:    &rd.EntityInfo.ShortTitle,
			LongTitle:     &rd.EntityInfo.LongTitle,
			Note:          &rd.EntityInfo.Note,
		})
		if err != nil {
			return errors.Trace(err)
		}

		// Delete existing contact info
		if _, err := dl.DeleteEntityContactsForEntityID(eID); err != nil {
			return errors.Trace(err)
		}

		// Build out the contacts we want to insert
		contacts := make([]*dal.EntityContact, len(rd.Contacts))
		for i, contact := range rd.Contacts {
			contactType, err := dal.ParseEntityContactType(directory.ContactType_name[int32(contact.ContactType)])
			if err != nil {
				return errors.Trace(err)
			}
			contacts[i] = &dal.EntityContact{
				EntityID:    eID,
				Type:        contactType,
				Value:       contact.Value,
				Provisioned: contact.Provisioned,
				Label:       contact.Label,
			}
		}

		// Insert the new set of contact records
		if err := dl.InsertEntityContacts(contacts); err != nil {
			return errors.Trace(err)
		}

		// Upsert any serialized entity contact info
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

		entity, err := dl.Entity(eID)
		if err != nil {
			return errors.Trace(err)
		}

		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.UpdateEntityResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) validateUpdateEntityRequest(rd *directory.UpdateEntityRequest) error {
	golog.Debugf("Entering server.server.validateUpdateEntityRequest: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.validateUpdateEntityRequest...") }()
	}
	eID, err := dal.ParseEntityID(rd.EntityID)
	if err != nil {
		return grpcErrorf(codes.InvalidArgument, "Unable to parse entity ID")
	}
	entity, err := s.dl.Entity(eID)
	if api.IsErrNotFound(err) {
		return grpcErrorf(codes.NotFound, err.Error())
	} else if err != nil {
		return grpcErrorf(codes.Internal, err.Error())
	}
	if entity.Type != dal.EntityTypeExternal && rd.EntityInfo.DisplayName == "" {
		return grpcErrorf(codes.InvalidArgument, "Display Name cannot be empty for non external entities")
	}
	return nil
}

func (s *server) ExternalIDs(ctx context.Context, rd *directory.ExternalIDsRequest) (out *directory.ExternalIDsResponse, err error) {
	golog.Debugf("Entering server.server.ExternalIDs: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.ExternalIDs... %+v", out) }()
	}
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
	if api.IsErrNotFound(err) {
		return false, nil
	}
	return true, errors.Trace(err)
}

func (s *server) CreateMembership(ctx context.Context, rd *directory.CreateMembershipRequest) (*directory.CreateMembershipResponse, error) {
	golog.Debugf("Entering server.server.CreateMembership: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CreateMembership...") }()
	}
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
	pbEntity, err := getPBEntity(s.dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &directory.CreateMembershipResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) LookupEntitiesByContact(ctx context.Context, rd *directory.LookupEntitiesByContactRequest) (out *directory.LookupEntitiesByContactResponse, err error) {
	golog.Debugf("Entering server.server.LookupEntitiesByContact: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.LookupEntitiesByContact... %+v", out) }()
	}
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
	entities, err := s.dl.Entities(entityIDs, statuses)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	pbEntities, err := getPBEntities(s.dl, entities, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), statuses)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &directory.LookupEntitiesByContactResponse{
		Entities: pbEntities,
	}, nil
}

func (s *server) LookupEntityDomain(ctx context.Context, in *directory.LookupEntityDomainRequest) (*directory.LookupEntityDomainResponse, error) {
	golog.Debugf("Entering server.LookupEntityDomain: %+v", in)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.LookupEntityDomain...") }()
	}

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
	if api.IsErrNotFound(errors.Cause(err)) {
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

	if err := s.dl.InsertEntityDomain(eID, strings.ToLower(in.Domain)); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &directory.CreateEntityDomainResponse{}, nil
}

func (s *server) CreateContact(ctx context.Context, rd *directory.CreateContactRequest) (out *directory.CreateContactResponse, err error) {
	golog.Debugf("Entering server.server.CreateContact: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CreateContact... %+v", out) }()
	}
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
		}); err != nil {
			return errors.Trace(err)
		}
		entity, err := dl.Entity(entityID)
		if err != nil {
			return errors.Trace(err)
		}
		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.CreateContactResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) CreateContacts(ctx context.Context, rd *directory.CreateContactsRequest) (out *directory.CreateContactsResponse, err error) {
	golog.Debugf("Entering server.server.CreateContacts: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CreateContacts... %+v", out) }()
	}
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
		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil)
		return errors.Trace(err)
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
	golog.Debugf("Entering server.server.UpdateContacts: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.UpdateContacts... %+v", out) }()
	}
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
				Type:  &contactType,
				Value: &c.Value,
				Label: &c.Label,
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
		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), nil)
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.UpdateContactsResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) DeleteContacts(ctx context.Context, rd *directory.DeleteContactsRequest) (out *directory.DeleteContactsResponse, err error) {
	golog.Debugf("Entering server.server.DeleteContacts: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.DeleteContacts... %+v", out) }()
	}
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
		pbEntity, err = getPBEntity(dl, entity, riEntityInformation(rd.RequestedInformation), riDepth(rd.RequestedInformation), []dal.EntityStatus{})
		return errors.Trace(err)
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &directory.DeleteContactsResponse{
		Entity: pbEntity,
	}, nil
}

func (s *server) DeleteEntity(ctx context.Context, rd *directory.DeleteEntityRequest) (out *directory.DeleteEntityResponse, err error) {
	golog.Debugf("Entering server.server.DeleteEntityRequest: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.DeleteEntityRequest... %+v", out) }()
	}
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

func transformEntityStatuses(ss []directory.EntityStatus) ([]dal.EntityStatus, error) {
	if len(ss) == 0 {
		return nil, nil
	}
	dss := make([]dal.EntityStatus, len(ss))
	for i, s := range ss {
		ds, err := dal.ParseEntityStatus(s.String())
		if err != nil {
			return nil, errors.Trace(err)
		}
		dss[i] = ds
	}
	return dss, nil
}

func transformExternalIDs(dExternalEntityIDs []*dal.ExternalEntityID) []*directory.ExternalID {
	pExternalID := make([]*directory.ExternalID, len(dExternalEntityIDs))
	for i, eID := range dExternalEntityIDs {
		pExternalID[i] = &directory.ExternalID{
			ID:       eID.ExternalID,
			EntityID: eID.EntityID.String(),
		}
	}
	return pExternalID
}

func getPBEntities(dl dal.DAL, dEntities []*dal.Entity, entityInformation []directory.EntityInformation, depth int64, statuses []dal.EntityStatus) ([]*directory.Entity, error) {
	pbEntities := make([]*directory.Entity, len(dEntities))
	for i, e := range dEntities {
		pbEntity, err := getPBEntity(dl, e, entityInformation, depth, statuses)
		if err != nil {
			return nil, errors.Trace(err)
		}
		pbEntities[i] = pbEntity
	}
	return pbEntities, nil
}

// Note: How we optimize this deep crawl is very likely to change
func getPBEntity(dl dal.DAL, dEntity *dal.Entity, entityInformation []directory.EntityInformation, depth int64, statuses []dal.EntityStatus) (*directory.Entity, error) {
	entity := dalEntityAsPBEntity(dEntity)
	if depth >= 0 {
		if hasRequestedInfo(entityInformation, directory.EntityInformation_MEMBERSHIPS) {
			memberships, err := getPBMemberships(dl, dEntity.ID, entityInformation, depth-1, statuses)
			if err != nil {
				return nil, errors.Trace(err)
			}
			entity.Memberships = memberships
		}
		if hasRequestedInfo(entityInformation, directory.EntityInformation_MEMBERS) {
			members, err := getPBMembers(dl, dEntity.ID, entityInformation, depth-1, statuses)
			if err != nil {
				return nil, errors.Trace(err)
			}
			entity.Members = members
		}
		if hasRequestedInfo(entityInformation, directory.EntityInformation_EXTERNAL_IDS) {
			externalIDs, err := getPBExternalIDs(dl, dEntity.ID)
			if err != nil {
				return nil, errors.Trace(err)
			}
			entity.ExternalIDs = externalIDs
		}
		if hasRequestedInfo(entityInformation, directory.EntityInformation_CONTACTS) {
			contacts, err := getPBEntityContacts(dl, dEntity.ID)
			if err != nil {
				return nil, errors.Trace(err)
			}
			entity.Contacts = contacts
		}
	}
	entity.IncludedInformation = entityInformation
	return entity, nil
}

func getPBMemberships(dl dal.DAL, entityID dal.EntityID, entityInformation []directory.EntityInformation, depth int64, statuses []dal.EntityStatus) ([]*directory.Entity, error) {
	memberships, err := dl.EntityMemberships(entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	entityIDs := make([]dal.EntityID, len(memberships))
	for i, membership := range memberships {
		entityIDs[i] = membership.TargetEntityID
	}
	entities, err := dl.Entities(entityIDs, statuses)
	if err != nil {
		return nil, errors.Trace(err)
	}
	pbEntities, err := getPBEntities(dl, entities, entityInformation, depth, statuses)
	return pbEntities, errors.Trace(err)
}

func getPBMembers(dl dal.DAL, entityID dal.EntityID, entityInformation []directory.EntityInformation, depth int64, statuses []dal.EntityStatus) ([]*directory.Entity, error) {
	members, err := dl.EntityMembers(entityID, statuses)
	if err != nil {
		return nil, errors.Trace(err)
	}
	pbEntities, err := getPBEntities(dl, members, entityInformation, depth, statuses)
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
		pbEntityContacts[i] = dalEntityContactAsPBContact(ec)
	}
	return pbEntityContacts, nil
}

func dalEntityContactAsPBContact(dEntityContact *dal.EntityContact) *directory.Contact {
	contact := &directory.Contact{}
	contact.Provisioned = dEntityContact.Provisioned
	contactType, ok := directory.ContactType_value[dEntityContact.Type.String()]
	if !ok {
		golog.Errorf("Unknown contact type %s when converting to PB format", dEntityContact.Type)
	}
	contact.ContactType = directory.ContactType(contactType)
	contact.Value = dEntityContact.Value
	contact.ID = dEntityContact.ID.String()
	contact.Label = dEntityContact.Label
	return contact
}

func dalEntityAsPBEntity(dEntity *dal.Entity) *directory.Entity {
	entity := &directory.Entity{
		ID:               dEntity.ID.String(),
		CreatedTimestamp: uint64(dEntity.Created.Unix()),
	}
	entityType, ok := directory.EntityType_value[dEntity.Type.String()]
	if !ok {
		golog.Errorf("Unknown entity type %s when converting to PB format", dEntity.Type)
	}
	entity.Type = directory.EntityType(entityType)
	entityStatus, ok := directory.EntityStatus_value[dEntity.Status.String()]
	if !ok {
		golog.Errorf("Unknown entity status %s when converting to PB format", dEntity.Status)
	}
	entity.LastModifiedTimestamp = uint64(dEntity.Modified.Unix())
	entity.Status = directory.EntityStatus(entityStatus)
	entity.Info = &directory.EntityInfo{
		FirstName:     dEntity.FirstName,
		MiddleInitial: dEntity.MiddleInitial,
		LastName:      dEntity.LastName,
		GroupName:     dEntity.GroupName,
		DisplayName:   dEntity.DisplayName,
		ShortTitle:    dEntity.ShortTitle,
		LongTitle:     dEntity.LongTitle,
		Note:          dEntity.Note,
	}
	return entity
}

// Note: Much letters. Many length. So convention.
func dalSerializedClientEntityContactAsPBSerializedClientEntityContact(dSerializedClientEntityContact *dal.SerializedClientEntityContact) *directory.SerializedClientEntityContact {
	serializedClientEntityContact := &directory.SerializedClientEntityContact{}
	serializedClientEntityContact.EntityID = dSerializedClientEntityContact.EntityID.String()
	platform, ok := directory.Platform_value[dSerializedClientEntityContact.Platform.String()]
	if !ok {
		golog.Errorf("Unknown platform %s when converting to PB format", dSerializedClientEntityContact.Platform)
	}
	serializedClientEntityContact.Platform = directory.Platform(platform)
	serializedClientEntityContact.SerializedEntityContact = dSerializedClientEntityContact.SerializedEntityContact
	return serializedClientEntityContact
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
