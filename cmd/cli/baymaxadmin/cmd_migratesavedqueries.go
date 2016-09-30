package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

type migrateSavedQueriesCmd struct {
	cnf          *config
	threadingCli threading.ThreadsClient
	directoryCli directory.DirectoryClient
	directoryDB  *sql.DB
}

func newMigrateSavedQueriesCmd(cnf *config) (command, error) {
	threadingCli, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}
	directoryCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	directoryDB, err := cnf.db("directory")
	if err != nil {
		return nil, err
	}
	return &migrateSavedQueriesCmd{
		cnf:          cnf,
		threadingCli: threadingCli,
		directoryCli: directoryCli,
		directoryDB:  directoryDB,
	}, nil
}

func (c *migrateSavedQueriesCmd) run(args []string) error {
	fs := flag.NewFlagSet("migratesavedqueries", flag.ExitOnError)
	flagEntityID := fs.String("entity_id", "", "optional entity ID instead of migrating all entities")
	flagRebuildAll := fs.Bool("rebuild", false, "rebuild all saved queries")
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

	for _, eid := range entityIDs {
		res, err := c.threadingCli.SavedQueries(ctx, &threading.SavedQueriesRequest{EntityID: eid})
		if err != nil {
			return errors.Trace(err)
		}

		var followersSQ *threading.SavedQuery
		for _, sq := range res.SavedQueries {
			if sq.Title == "Following" {
				followersSQ = sq
			}
		}

		// Create followers tab if missing

		if followersSQ == nil {
			fmt.Printf("Creating followers saved query for entity %s\n", eid)
			if _, err := c.threadingCli.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
				Type:                 threading.SAVED_QUERY_TYPE_NORMAL,
				EntityID:             eid,
				Title:                "Following",
				Ordinal:              5000,
				NotificationsEnabled: true,
				Query:                &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_FOLLOWING}}}},
			}); err != nil {
				golog.Errorf("Failed to create saved query 'Following': %s", err)
			}
		}

		if *flagRebuildAll {
			for _, sq := range res.SavedQueries {
				if sq.Type == threading.SAVED_QUERY_TYPE_NORMAL {
					fmt.Printf("Rebuilding saved query %s '%s' (%d)\n", sq.ID, sq.Title, sq.Total)
					if _, err := c.threadingCli.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
						SavedQueryID: sq.ID,
						ForceRebuild: true,
					}); err != nil {
						golog.Errorf("Failed to force rebuild of saved query %s for entity %s: %s", sq.ID, eid, err)
					}
				}
			}
		}

	}

	return nil
}
