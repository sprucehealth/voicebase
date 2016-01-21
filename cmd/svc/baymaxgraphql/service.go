package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/libs/conc"
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
	mediaSigner  *media.Signer
}

func (s *service) hydrateThreadTitles(ctx context.Context, threads []*thread) error {
	// TODO: this done one request per thread. ideally the directory service would have a bulk lookup
	p := conc.NewParallel()
	for _, t := range threads {
		if t.PrimaryEntityID == "" {
			// TODO: not sure what this should be for internal threads (ones without a primary entity ID)
			t.Title = "Internal"
			continue
		}
		// Create a reference to thread since the loop variable will change underneath
		thread := t
		p.Go(func() error {
			res, err := s.directory.LookupEntities(ctx,
				&directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: thread.PrimaryEntityID,
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth: 0,
						EntityInformation: []directory.EntityInformation{
							directory.EntityInformation_CONTACTS,
						},
					},
				})
			if err != nil {
				return errors.Trace(err)
			}
			if len(res.Entities) != 1 {
				return errors.Trace(fmt.Errorf("lookup entities returned %d results for %s, expected 1", len(res.Entities), thread.PrimaryEntityID))
			}
			thread.Title = threadTitleForEntity(res.Entities[0])
			return nil
		})
	}
	return p.Wait()
}

func (s *service) entityForAccountID(ctx context.Context, orgID, accountID string) (*directory.Entity, error) {
	// TODO: should use a cache for this
	res, err := s.directory.LookupEntities(ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: accountID,
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
