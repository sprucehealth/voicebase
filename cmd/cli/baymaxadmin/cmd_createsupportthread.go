package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

const (
	supportThreadTitle    = "Spruce Support"
	teamSpruceInitialText = `Use this conversation to chat with directly with the team at Spruce 😀.

We’re here 7am-7pm PST to answer any questions you have - big or small. Feel free to drop us a line at any time!`
)

type createSupportThreadCmd struct {
	cnf          *config
	threadingCli threading.ThreadsClient
	directoryCli directory.DirectoryClient
	threadingDB  *sql.DB
}

func newCreateSupportThreadCmd(cnf *config) (command, error) {
	threadingCli, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}

	directorCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	threadingDB, err := cnf.db("threading")
	if err != nil {
		return nil, err
	}

	return &createSupportThreadCmd{
		cnf:          cnf,
		threadingCli: threadingCli,
		directoryCli: directorCli,
		threadingDB:  threadingDB,
	}, nil
}

func (c *createSupportThreadCmd) run(args []string) error {
	fs := flag.NewFlagSet("createsupporthread", flag.ExitOnError)
	flagEntityID := fs.String("org_entity_id", "", "org entity for which to create support thread")
	spruceOrgID := fs.String("spruce_org_id", "", "org_id for the spruce support organization")
	ctx := context.Background()

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *flagEntityID == "" {
		return errors.Errorf("org_entity_id required")
	}

	if *spruceOrgID == "" {
		return errors.Errorf("spruce_org_id required")
	}

	// ensure that the entity specified is that of an organization
	entity, err := directory.SingleEntity(ctx, c.directoryCli, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: *flagEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{
			directory.EntityType_ORGANIZATION,
		},
	})
	if err != nil {
		return errors.Trace(err)
	} else if entity.Type != directory.EntityType_ORGANIZATION {
		return errors.Errorf("Expected organization but got entity of type %s for %s", entity.Type, entity.ID)
	}

	// do nothing if the support thread already exists for the organization
	var id string
	if err := c.threadingDB.QueryRow(`SELECT id FROM threads WHERE type='SUPPORT' AND organization_id=?`, *flagEntityID).Scan(&id); err != sql.ErrNoRows && err != nil {
		return errors.Trace(err)
	} else if err == nil {
		fmt.Println("Support thread already exists for " + *flagEntityID)
		return nil
	}

	tsEnt1Res, err := c.directoryCli.CreateEntity(ctx, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			GroupName:   supportThreadTitle,
			DisplayName: supportThreadTitle,
		},
		Type: directory.EntityType_SYSTEM,
		InitialMembershipEntityID: *flagEntityID,
	})
	if err != nil {
		return errors.Trace(err)
	}

	remoteSupportThreadTitle := fmt.Sprintf("%s (%s)", supportThreadTitle, entity.Info.DisplayName)
	tsEnt2Res, err := c.directoryCli.CreateEntity(ctx, &directory.CreateEntityRequest{
		EntityInfo: &directory.EntityInfo{
			GroupName:   remoteSupportThreadTitle,
			DisplayName: remoteSupportThreadTitle,
		},
		Type: directory.EntityType_SYSTEM,
		InitialMembershipEntityID: *spruceOrgID,
	})
	if err != nil {
		return errors.Trace(err)
	}

	_, err = c.threadingCli.CreateLinkedThreads(ctx, &threading.CreateLinkedThreadsRequest{
		Organization1ID:      *flagEntityID,
		Organization2ID:      *spruceOrgID,
		PrimaryEntity1ID:     tsEnt1Res.Entity.ID,
		PrimaryEntity2ID:     tsEnt2Res.Entity.ID,
		PrependSenderThread1: false,
		PrependSenderThread2: true,
		Summary:              supportThreadTitle + ": " + teamSpruceInitialText[:128],
		Text:                 teamSpruceInitialText,
		Type:                 threading.THREAD_TYPE_SUPPORT,
		SystemTitle1:         supportThreadTitle,
		SystemTitle2:         remoteSupportThreadTitle,
	})
	if err != nil {
		return errors.Errorf("Failed to create linked support threads for org %s: %s", *flagEntityID, err)
	}

	return nil
}
