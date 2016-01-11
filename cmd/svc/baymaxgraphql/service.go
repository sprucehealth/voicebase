package main

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type service struct {
	auth         auth.AuthClient
	directory    directory.DirectoryClient
	threading    threading.ThreadsClient
	exComms      excomms.ExCommsClient
	notification notification.Client
}

func (s *service) entityForAccountID(ctx context.Context, orgID, accountID string) (*directory.Entity, error) {
	// TODO: should use a cache for this
	res, err := s.directory.LookupEntities(ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: accountIDType + ":" + accountID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					// TODO: don't always need contacts
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	for _, e := range res.Entities {
		for _, e2 := range e.GetMemberships() {
			if e2.Type == directory.EntityType_ORGANIZATION && e2.ID == orgID {
				return e, nil
			}
		}
	}
	return nil, nil
}

func (s *service) entity(ctx context.Context, entityID string) (*directory.Entity, error) {
	// TODO: should use a cache for this
	res, err := s.directory.LookupEntities(ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: entityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					// TODO: don't always need contacts
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	for _, e := range res.Entities {
		return e, nil
	}
	return nil, nil
}
