package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/directory/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
)

// dalEntityAsPBEntity transforms a dal entity contact to a svc entity contact.
// the dal entity contact must not be used after this call.
func transformEntityContactToResponse(dEntityContact *dal.EntityContact) *directory.Contact {
	contact := &directory.Contact{
		Provisioned: dEntityContact.Provisioned,
		Value:       dEntityContact.Value,
		ID:          dEntityContact.ID.String(),
		Label:       dEntityContact.Label,
	}
	contactType, ok := directory.ContactType_value[dEntityContact.Type.String()]
	if !ok {
		golog.Errorf("Unknown contact type %s when converting to PB format", dEntityContact.Type)
	}
	contact.ContactType = directory.ContactType(contactType)
	dEntityContact.Recycle()
	return contact
}

// dalEntityAsPBEntity transforms a dal entity to a svc entity.
// the dal entity must not be used after this call.
func transformEntityToResponse(dEntity *dal.Entity) (*directory.Entity, error) {
	entity := &directory.Entity{
		ID:                    dEntity.ID.String(),
		CreatedTimestamp:      uint64(dEntity.Created.Unix()),
		LastModifiedTimestamp: uint64(dEntity.Modified.Unix()),
		AccountID:             dEntity.AccountID,
		ImageMediaID:          dEntity.ImageMediaID,
	}
	entityType, ok := directory.EntityType_value[dEntity.Type.String()]
	if !ok {
		return nil, fmt.Errorf("unknown entity type %s when converting to PB format", dEntity.Type)
	}
	entity.Type = directory.EntityType(entityType)
	entityStatus, ok := directory.EntityStatus_value[dEntity.Status.String()]
	if !ok {
		return nil, fmt.Errorf("unknown entity status %s when converting to PB format", dEntity.Status)
	}

	entity.Status = directory.EntityStatus(entityStatus)
	entity.Info = transformEntityToEntityInfoResponse(dEntity)
	dEntity.Recycle()
	return entity, nil
}

func transformEntityToEntityInfoResponse(de *dal.Entity) *directory.EntityInfo {
	var entityGender directory.EntityInfo_Gender
	if de.Gender != nil {
		entityGender = directory.EntityInfo_Gender(directory.EntityInfo_Gender_value[de.Gender.String()])
	}
	var dob *directory.Date
	if de.DOB != nil {
		dob = &directory.Date{
			Month: uint32(de.DOB.Month),
			Day:   uint32(de.DOB.Day),
			Year:  uint32(de.DOB.Year),
		}
	}
	return &directory.EntityInfo{
		FirstName:     de.FirstName,
		MiddleInitial: de.MiddleInitial,
		LastName:      de.LastName,
		GroupName:     de.GroupName,
		DisplayName:   de.DisplayName,
		ShortTitle:    de.ShortTitle,
		LongTitle:     de.LongTitle,
		Gender:        entityGender,
		DOB:           dob,
		Note:          de.Note,
	}
}

// Note: Much letters. Many length. So convention.
func transformSerializedClientEntityContactToResponse(dSerializedClientEntityContact *dal.SerializedClientEntityContact) *directory.SerializedClientEntityContact {
	serializedClientEntityContact := &directory.SerializedClientEntityContact{
		EntityID:                dSerializedClientEntityContact.EntityID.String(),
		SerializedEntityContact: dSerializedClientEntityContact.SerializedEntityContact,
	}
	platform, ok := directory.Platform_value[dSerializedClientEntityContact.Platform.String()]
	if !ok {
		golog.Errorf("Unknown platform %s when converting to PB format", dSerializedClientEntityContact.Platform)
	}
	serializedClientEntityContact.Platform = directory.Platform(platform)
	return serializedClientEntityContact
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

func transformEntityTypes(types []directory.EntityType) ([]dal.EntityType, error) {
	transformedTypes := make([]dal.EntityType, len(types))
	for i, t := range types {
		parsedType, err := dal.ParseEntityType(t.String())
		if err != nil {
			return nil, errors.Trace(err)
		}
		transformedTypes[i] = parsedType
	}

	return transformedTypes, nil
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

func transformEntityProfileToResponse(p *dal.EntityProfile) *directory.Profile {
	rP := &directory.Profile{
		ID:                    p.ID.String(),
		EntityID:              p.EntityID.String(),
		Sections:              p.Sections.Sections,
		LastModifiedTimestamp: uint64(p.Modified.Unix()),
	}
	return rP
}
