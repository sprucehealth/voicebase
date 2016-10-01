package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification"
)

type zeroBadgesCmd struct {
	cnf             *config
	directoryCli    directory.DirectoryClient
	directoryDB     *sql.DB
	notificationCli notification.Client
}

func newZeroBadgesCmd(cnf *config) (command, error) {
	directoryCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	directoryDB, err := cnf.db("directory")
	if err != nil {
		return nil, err
	}
	eSQS, err := cnf.sqsClient()
	if err != nil {
		return nil, errors.Trace(err)
	}
	if cnf.NotificationsSQSURL == "" {
		return nil, errors.New("NotificationsSQSURL required")
	}
	notificationClient := notification.NewClient(eSQS, &notification.ClientConfig{
		SQSNotificationURL: cnf.NotificationsSQSURL,
	})
	return &zeroBadgesCmd{
		cnf:             cnf,
		directoryCli:    directoryCli,
		directoryDB:     directoryDB,
		notificationCli: notificationClient,
	}, nil
}

func (c *zeroBadgesCmd) run(args []string) error {
	fs := flag.NewFlagSet("zerobadges", flag.ExitOnError)
	flagEntityID := fs.String("entity_id", "", "optional entity ID instead of migrating all entities")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fs.Parse(args)

	var entityIDs []string

	ctx := context.Background()
	if *flagEntityID != "" {
		// Verify that the entity is internal
		res, err := c.directoryCli.LookupEntities(ctx, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: *flagEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{},
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{
				directory.EntityType_INTERNAL,
			},
		})
		if err != nil {
			return errors.Errorf("Failed to lookup entity ID %s: %s", *flagEntityID, err)
		}
		if len(res.Entities) == 0 {
			return errors.Errorf("No entity found for entity ID %s", *flagEntityID)
		}
		if e := res.Entities[0]; e.Type != directory.EntityType_INTERNAL {
			return errors.Errorf("Entity is %s, expected %s", e.Type, directory.EntityType_INTERNAL)
		}
		entityIDs = []string{res.Entities[0].ID}
	} else {
		var err error
		entityIDs, err = internalEntityIDs(c.directoryDB)
		if err != nil {
			return errors.Trace(err)
		}
	}

	const batchSize = 200
	for len(entityIDs) > 0 {
		batch := entityIDs
		if len(batch) > batchSize {
			batch = entityIDs[:batchSize]
		}
		entityIDs = entityIDs[len(batch):]
		if err := c.notificationCli.SendNotification(&notification.Notification{
			EntitiesToNotify: batch,
			ForceZeroBadge:   true,
			CollapseKey:      "zero_badge",
			DedupeKey:        "zero_badge",
			Type:             notification.BadgeCount,
		}); err != nil {
			fmt.Printf("Failed to notify entities %s\n", strings.Join(batch, ","))
		}
	}

	return nil
}
