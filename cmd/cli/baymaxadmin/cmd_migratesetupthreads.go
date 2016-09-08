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

type migrateSetupThreadsCmd struct {
	cnf          *config
	threadingCli threading.ThreadsClient
	directoryCli directory.DirectoryClient
	threadingDB  *sql.DB
}

func newMigrateSetupThreadsCmd(cnf *config) (command, error) {
	threadingCli, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}
	directoryCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	threadingDB, err := cnf.db("threading")
	if err != nil {
		return nil, err
	}
	return &migrateSetupThreadsCmd{
		cnf:          cnf,
		threadingCli: threadingCli,
		directoryCli: directoryCli,
		threadingDB:  threadingDB,
	}, nil
}

func (c *migrateSetupThreadsCmd) run(args []string) error {
	fs := flag.NewFlagSet("migratesetupthreads", flag.ExitOnError)
	flagOrgID := fs.String("org_id", "", "optional organization ID instead of migrating all orgs without setup threads")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fs.Parse(args)

	var orgIDs []string

	ctx := context.Background()
	if *flagOrgID != "" {
		res, err := c.directoryCli.LookupEntities(ctx, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: *flagOrgID,
			},
		})
		if err != nil {
			return fmt.Errorf("Failed to lookup org entity: %s", err)
		}
		if len(res.Entities) == 0 {
			return errors.New("No entity found for org ID")
		}
		if e := res.Entities[0]; e.Type != directory.EntityType_ORGANIZATION {
			return fmt.Errorf("Entity is %s, expected %s", e.Type, directory.EntityType_ORGANIZATION)
		}
		orgIDs = []string{*flagOrgID}
	} else {
		var err error
		orgIDs, err = orgIDsWithoutSetupThread(c.threadingDB)
		if err != nil {
			return err
		}
	}

	for _, orgID := range orgIDs {
		fmt.Printf("Creating thread for org %s\n", orgID)
		entRes, err := c.directoryCli.CreateEntity(ctx, &directory.CreateEntityRequest{
			EntityInfo: &directory.EntityInfo{
				GroupName: "Setup Assistant",
			},
			Type: directory.EntityType_SYSTEM,
			InitialMembershipEntityID: orgID,
		})
		if err != nil {
			return fmt.Errorf("Failed to create entity for setup thread for org %s: %s", orgID, err)
		}
		_, err = c.threadingCli.CreateOnboardingThread(ctx, &threading.CreateOnboardingThreadRequest{
			OrganizationID:  orgID,
			PrimaryEntityID: entRes.Entity.ID,
		})
		if err != nil {
			return fmt.Errorf("Failed to create onboarding thread for org %s: %s", orgID, err)
		}
	}

	return nil
}

func orgIDsWithoutSetupThread(db *sql.DB) ([]string, error) {
	// NOTE: only using the threading db since it makes it easier and all orgs should have at least 1 thread so should show up there anyway

	// Get a list of all organizations IDs
	rows, err := db.Query(`SELECT distinct organization_id FROM threads`)
	if err != nil {
		return nil, fmt.Errorf("Failed to get list of org IDs: %s", err)
	}
	defer rows.Close()
	var orgIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("Failed to scan org ID: %s", err)
		}
		orgIDs = append(orgIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Failed to get list of org IDs: %s", err)
	}

	// Get a list of organization IDs that already have a setup thread
	rows, err = db.Query(`SELECT distinct organization_id FROM threads WHERE type = ? AND deleted = ?`, "SETUP", false)
	if err != nil {
		return nil, fmt.Errorf("Failed to get list of org IDs with setup thread: %s", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("Failed to scan org ID: %s", err)
		}
		// Inefficient but whatevs
		for i, oid := range orgIDs {
			if oid == id {
				orgIDs[i] = orgIDs[len(orgIDs)-1]
				orgIDs = orgIDs[:len(orgIDs)-1]
				break
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Failed to get list of org IDs with setup thread: %s", err)
	}

	return orgIDs, nil
}
