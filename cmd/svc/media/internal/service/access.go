package service

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

func (s *service) CanAccess(ctx context.Context, mediaID dal.MediaID, accountID string) error {
	media, err := s.dal.Media(mediaID)
	if errors.Cause(err) == dal.ErrNotFound {
		// For legacy media we don't have info for, allow access, rely just on auth for now
		return nil
	} else if err != nil {
		return err
	}
	// Non prod hack for allowing old media
	// TODO: Remove all this crud by cleaning up preprod eventually
	if !environment.IsProd() && media.OwnerID == "TODO" {
		return nil
	}
	switch media.OwnerType {
	case dal.MediaOwnerTypeAccount:
		if media.OwnerID != accountID {
			return ErrAccessDenied
		}
		return nil
	case dal.MediaOwnerTypeEntity:
		return s.canAccessEntityMedia(ctx, media.OwnerID, accountID)
	case dal.MediaOwnerTypeOrganization:
		return s.canAccessOrganizationMedia(ctx, media.OwnerID, accountID)
	case dal.MediaOwnerTypeThread:
		return s.canAccessThreadMedia(ctx, media.OwnerID, accountID)
	}
	return fmt.Errorf("Unsupported Media Owner Type: %s", media.OwnerType)
}

func (s *service) entitiesForAccountID(ctx context.Context, accountID string) ([]*directory.Entity, error) {
	resp, err := s.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
			AccountID: accountID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
	})
	if err != nil {
		return nil, err
	}
	return resp.Entities, nil
}

func (s *service) canAccessEntityMedia(ctx context.Context, entityID, accountID string) error {
	ents, err := s.entitiesForAccountID(ctx, accountID)
	if err != nil {
		return err
	}
	for _, ent := range ents {
		if ent.ID == entityID {
			return nil
		}
	}
	return ErrAccessDenied
}

func (s *service) canAccessOrganizationMedia(ctx context.Context, organizationID, accountID string) error {
	resp, err := s.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
			AccountID: accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return err
	}
	for _, ent := range resp.Entities {
		for _, mem := range ent.Memberships {
			if mem.ID == organizationID {
				return nil
			}
		}
	}
	return ErrAccessDenied
}

func (s *service) canAccessThreadMedia(ctx context.Context, threadID, accountID string) error {
	parallel := conc.NewParallel()
	var threadMembers []*threading.Member
	var accountEntities []*directory.Entity
	tResp, err := s.threads.Thread(ctx, &threading.ThreadRequest{
		ThreadID: threadID,
	})
	if err != nil {
		return err
	}
	// If this is a non team thread then just do an org check
	if tResp.Thread.Type != threading.ThreadType_TEAM {
		return s.canAccessOrganizationMedia(ctx, tResp.Thread.OrganizationID, accountID)
	}
	parallel.Go(func() error {
		resp, err := s.threads.ThreadMembers(ctx, &threading.ThreadMembersRequest{
			ThreadID: threadID,
		})
		if err != nil {
			return err
		}
		threadMembers = resp.Members
		return nil
	})
	parallel.Go(func() error {
		ents, err := s.entitiesForAccountID(ctx, accountID)
		if err != nil {
			return err
		}
		accountEntities = ents
		return nil
	})
	if err := parallel.Wait(); err != nil {
		return err
	}
	for _, threadMember := range threadMembers {
		for _, accountEnt := range accountEntities {
			if threadMember.EntityID == accountEnt.ID {
				return nil
			}
		}
	}
	return ErrAccessDenied
}
