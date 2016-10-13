package externalmsg

import (
	"strings"

	"context"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
)

func determineAccountIDFromEntityExternalID(ent *directory.Entity) string {
	for _, externalID := range ent.ExternalIDs {
		if strings.HasPrefix(externalID, auth.AccountIDPrefix) {
			return externalID
		}
	}
	return ""
}

func determineDisplayName(channelID string, contactType directory.ContactType, entity *directory.Entity) string {
	fromName := channelID
	if entity.Info != nil && entity.Info.DisplayName != "" {
		return entity.Info.DisplayName
	} else if contactType == directory.ContactType_PHONE {
		formattedPhone, err := phone.Format(fromName, phone.Pretty)
		if err == nil {
			return formattedPhone
		}
	}
	return fromName
}

func lookupEntities(ctx context.Context, entityID string, dir directory.DirectoryClient) ([]*directory.Entity, error) {
	res, err := dir.LookupEntities(
		ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: entityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res.Entities, nil
}

func lookupEntitiesByContact(ctx context.Context, contactValue string, dir directory.DirectoryClient) (*directory.LookupEntitiesByContactResponse, error) {
	res, err := dir.LookupEntitiesByContact(
		ctx,
		&directory.LookupEntitiesByContactRequest{
			ContactValue: contactValue,
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
					directory.EntityInformation_EXTERNAL_IDS,
				},
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return res, nil
}

func determineOrganization(entity *directory.Entity) *directory.Entity {
	if entity.Type == directory.EntityType_ORGANIZATION {
		return entity
	}

	for _, m := range entity.Memberships {
		if m.Type == directory.EntityType_ORGANIZATION {
			return m
		}
	}

	return nil
}

func determineProviderOrOrgEntity(res *directory.LookupEntitiesByContactResponse, value string) *directory.Entity {
	if res == nil {
		return nil
	}
	for _, entity := range res.Entities {
		switch entity.Type {
		case directory.EntityType_ORGANIZATION, directory.EntityType_INTERNAL:
		case directory.EntityType_EXTERNAL:
			continue
		}
		for _, c := range entity.Contacts {

			if strings.EqualFold(c.Value, value) {
				return entity
			}
		}
	}
	return nil
}

func determineExternalEntities(res *directory.LookupEntitiesByContactResponse, organizationID string) []*directory.Entity {
	if res == nil {
		return nil
	}

	externalEntities := make([]*directory.Entity, 0, len(res.Entities))
	for _, entity := range res.Entities {
		if entity.Type != directory.EntityType_EXTERNAL {
			continue
		}
		// if entity is external, determine membership to the specified organization.
		for _, m := range entity.Memberships {
			if m.Type == directory.EntityType_ORGANIZATION && m.ID == organizationID {
				externalEntities = append(externalEntities, entity)
			}
		}
	}
	return externalEntities
}
