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

type migrateSavedQueriesCmd struct {
	cnf          *config
	threadingCli threading.ThreadsClient
	directoryCli directory.DirectoryClient
	directoryDB  *sql.DB
}

func newmigrateSavedQueriesCmd(cnf *config) (command, error) {
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
		// Unmigrated entities have 1 saved query with ordinal of 0
		if len(res.SavedQueries) != 1 || res.SavedQueries[0].Ordinal != 0 {
			// Rebuild saved queries with no threads to be safe
			for _, sq := range res.SavedQueries {
				if sq.Total == 0 {
					fmt.Printf("Rebuilding saved query %s '%s'\n", sq.ID, sq.Title)
					for _, sq := range res.SavedQueries {
						if _, err := c.threadingCli.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
							SavedQueryID: sq.ID,
							ForceRebuild: true,
						}); err != nil {
							return errors.Errorf("Failed to force rebuild of saved query %s for entity %s: %s", sq.ID, eid, err)
						}
					}
				}
			}
			continue
		}
		fmt.Printf("Migration entity %s\n", eid)
		sq := res.SavedQueries[0]
		if _, err := c.threadingCli.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
			SavedQueryID: sq.ID,
			Title:        "All",
			Query:        &threading.Query{},
			Ordinal:      1000,
			ForceRebuild: true,
		}); err != nil {
			return errors.Errorf("Failed to update saved query %s for entity %s: %s", sq.ID, eid, err)
		}
		if _, err := c.threadingCli.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
			EntityID: eid,
			Title:    "Patient",
			Query:    &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}}}},
			Ordinal:  2000,
		}); err != nil {
			golog.Errorf("Failed to create saved query 'Patient': %s", err)
		}
		if _, err := c.threadingCli.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
			EntityID: eid,
			Title:    "Team",
			Query:    &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_TEAM}}}},
			Ordinal:  3000,
		}); err != nil {
			golog.Errorf("Failed to create saved query 'Team': %s", err)
		}
		if _, err := c.threadingCli.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
			EntityID: eid,
			Title:    "@Pages",
			Query:    &threading.Query{Expressions: []*threading.Expr{{Value: &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_UNREAD_REFERENCE}}}},
			Ordinal:  4000,
		}); err != nil {
			golog.Errorf("Failed to create saved query '@Pages': %s", err)
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
