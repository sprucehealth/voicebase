package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

type rebuildSavedQueriesCmd struct {
	cnf          *config
	threadingCli threading.ThreadsClient
	directoryCli directory.DirectoryClient
	directoryDB  *sql.DB
}

func newRebuildSavedQueriesCmd(cnf *config) (command, error) {
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
	return &rebuildSavedQueriesCmd{
		cnf:          cnf,
		threadingCli: threadingCli,
		directoryCli: directoryCli,
		directoryDB:  directoryDB,
	}, nil
}

func (c *rebuildSavedQueriesCmd) run(args []string) error {
	fs := flag.NewFlagSet("rebuildsavedqueries", flag.ExitOnError)
	flagAll := fs.Bool("all", false, "Migrate all saved queries for all entities")
	flagEntityID := fs.String("entity_id", "", "Entity ID for whom to migrate saved queries")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fs.Parse(args)

	var entityIDs []string

	ctx := context.Background()
	if !*flagAll {
		// Verify that the entity is internal
		res, err := c.directoryCli.LookupEntities(ctx, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_EntityID{
				EntityID: *flagEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERS,
				},
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{
				directory.EntityType_INTERNAL,
				directory.EntityType_ORGANIZATION,
			},
			ChildTypes: []directory.EntityType{
				directory.EntityType_INTERNAL,
			},
		})
		if err != nil {
			return errors.Errorf("Failed to lookup entity ID %s: %s", *flagEntityID, err)
		}
		if len(res.Entities) == 0 {
			return errors.Errorf("No entity found for entity ID %s", *flagEntityID)
		}
		e := res.Entities[0]
		switch e.Type {
		case directory.EntityType_INTERNAL:
			entityIDs = []string{e.ID}
		case directory.EntityType_ORGANIZATION:
			for _, em := range e.Members {
				entityIDs = append(entityIDs, em.ID)
			}
		default:
			return errors.Errorf("Entity is %s, expected %s", e.Type, directory.EntityType_INTERNAL)
		}
	} else {
		var err error
		entityIDs, err = internalEntityIDs(c.directoryDB)
		if err != nil {
			return errors.Trace(err)
		}
	}

	for _, eid := range entityIDs {
		fmt.Printf("Rebuilding saved queries for entity %s\n", eid)
		res, err := c.threadingCli.SavedQueries(ctx, &threading.SavedQueriesRequest{EntityID: eid})
		if err != nil {
			return errors.Trace(err)
		}
		for _, sq := range res.SavedQueries {
			if sq.Type == threading.SAVED_QUERY_TYPE_NORMAL {
				fmt.Printf("\t%s %s (%d)\n", sq.ID, sq.ShortTitle, sq.Total)
				if _, err := c.threadingCli.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
					SavedQueryID: sq.ID,
					ForceRebuild: true,
				}); err != nil {
					golog.Errorf("Failed to force rebuild of saved query %s for entity %s: %s", sq.ID, eid, err)
				}
			}
		}
	}

	return nil
}

func internalEntityIDs(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT id FROM entity WHERE type = ? ORDER BY id`, "INTERNAL")
	if err != nil {
		return nil, fmt.Errorf("Failed to get list of org IDs: %s", err)
	}
	defer rows.Close()
	var entityIDs []string
	for rows.Next() {
		id := model.ObjectID{Prefix: "entity_"}
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("Failed to scan org ID: %s", err)
		}
		entityIDs = append(entityIDs, id.String())
	}
	return entityIDs, errors.Trace(rows.Err())
}
