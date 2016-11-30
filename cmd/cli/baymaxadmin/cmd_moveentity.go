package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

type moveEntityCmd struct {
	cnf          *config
	dirCli       directory.DirectoryClient
	dirDB        *sql.DB
	threadingCli threading.ThreadsClient
}

func newMoveEntityCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	dirDB, err := cnf.directoryDB()
	if err != nil {
		return nil, err
	}
	threadingCli, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}
	return &moveEntityCmd{
		cnf:          cnf,
		dirCli:       dirCli,
		dirDB:        dirDB,
		threadingCli: threadingCli,
	}, nil
}

func (c *moveEntityCmd) run(args []string) error {
	fs := flag.NewFlagSet("moveentity", flag.ExitOnError)
	entityID := fs.String("entity_id", "", "ID of the entity that has the contact")
	newOrganizationID := fs.String("new_org_id", "", "ID of the organization where to move the entity")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *entityID == "" {
		*entityID = prompt(scn, "Entity ID: ")
	}
	if *entityID == "" {
		return errors.New("Entity ID required")
	}

	if *newOrganizationID == "" {
		*newOrganizationID = prompt(scn, "Organization entity ID: ")
	}
	if *newOrganizationID == "" {
		return errors.New("Organization entity ID required")
	}

	ctx := context.Background()

	fmt.Printf("Moving entity:\n\n")
	entity, err := lookupAndDisplayEntity(ctx, c.dirCli, *entityID, []directory.EntityInformation{
		directory.EntityInformation_CONTACTS,
		directory.EntityInformation_MEMBERSHIPS,
		directory.EntityInformation_EXTERNAL_IDS,
	})
	if err != nil {
		return err
	}
	if entity.Type != directory.EntityType_INTERNAL {
		return errors.Errorf("Can only move %s entities, not %s", directory.EntityType_INTERNAL, entity.Type)
	}
	if entity.Status != directory.EntityStatus_ACTIVE {
		return errors.Errorf("Expected original entity to be %s, got %s", directory.EntityStatus_ACTIVE, entity.Status)
	}

	fmt.Printf("\nFrom organization:\n\n")
	var oldOrgID string
	for _, em := range entity.Memberships {
		if em.Type == directory.EntityType_ORGANIZATION {
			oldOrgID = em.ID
			break
		}
	}
	if oldOrgID == "" {
		return errors.New("Entity is not a member of any existing organizations")
	}
	oldOrg, err := lookupAndDisplayEntity(ctx, c.dirCli, oldOrgID, []directory.EntityInformation{directory.EntityInformation_CONTACTS})
	if err != nil {
		return err
	}
	// Sanity check, we already selected specifically an org ID above
	if oldOrg.Type != directory.EntityType_ORGANIZATION {
		return errors.Errorf("Can only move an entity from an organization, not %s", oldOrg.Type)
	}

	fmt.Printf("\nTo organization:\n\n")
	newOrg, err := lookupAndDisplayEntity(ctx, c.dirCli, *newOrganizationID, []directory.EntityInformation{directory.EntityInformation_CONTACTS})
	if err != nil {
		return err
	}
	if newOrg.Type != directory.EntityType_ORGANIZATION {
		return errors.Errorf("Can only move an entity into an organization, not %s", newOrg.Type)
	}

	// TODO: automate the phone number validation
	fmt.Printf("\nNOTE: Make sure to verify that an invite was sent by someone in the new organization to a phone number matching the entity's contacts.\n\n")

	fmt.Println()
	if strings.ToLower(prompt(scn, "Move entity [y/N]? ")) != "y" {
		return nil
	}

	fmt.Println()
	if strings.ToLower(prompt(scn, "Did you verify an invite was sent and phone number matches [y/N]? ")) != "y" {
		return nil
	}

	// Create a new entity with the same information as the original
	ceReq := &directory.CreateEntityRequest{
		Type:                      entity.Type,
		ExternalID:                entity.AccountID,
		AccountID:                 entity.AccountID,
		InitialMembershipEntityID: newOrg.ID,
		Contacts:                  entity.Contacts,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
			},
		},
		EntityInfo: entity.Info,
	}
	ceRes, err := c.dirCli.CreateEntity(ctx, ceReq)
	if err != nil {
		return errors.Errorf("Failed to create new entity: %s", err)
	}
	fmt.Printf("\nCreated new entity in new org:\n\n")
	newEntity := ceRes.Entity
	displayEntity("", newEntity)

	// Create new default saved query

	savedQueryTemplatesRes, err := c.threadingCli.SavedQueryTemplates(ctx, &threading.SavedQueryTemplatesRequest{
		EntityID: newOrg.ID,
	})
	if err != nil {
		return errors.Errorf("Failed to get saved query templates for org %s : %s", newOrg.ID, err)
	}

	for _, savedQueryTemplate := range savedQueryTemplatesRes.SavedQueries {

		if _, err := c.threadingCli.CreateSavedQuery(ctx, &threading.CreateSavedQueryRequest{
			EntityID:             newEntity.ID,
			ShortTitle:           savedQueryTemplate.ShortTitle,
			LongTitle:            savedQueryTemplate.LongTitle,
			Description:          savedQueryTemplate.Description,
			Query:                savedQueryTemplate.Query,
			Ordinal:              savedQueryTemplate.Ordinal,
			NotificationsEnabled: savedQueryTemplate.NotificationsEnabled,
			Hidden:               savedQueryTemplate.Hidden,
			Type:                 savedQueryTemplate.Type,
		}); err != nil {
			golog.Errorf("Failed to create saved query %s : %s", savedQueryTemplate.ShortTitle, err)
		}
	}

	// Delete the old entity (really updates the status so we don't lose anything here)
	if _, err := c.dirCli.DeleteEntity(ctx, &directory.DeleteEntityRequest{EntityID: entity.ID}); err != nil {
		return errors.Errorf("Failed to delete old entity: %s", err)
	}

	// Remove external ID association of the account with the old entity
	// TODO: implement this through the directory service interface rather than directory DB access
	oldEntityDBID, err := decodeID(entity.ID)
	if err != nil {
		return err
	}
	tx, err := c.dirDB.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM external_entity_id WHERE entity_id = ? AND external_id = ?`, oldEntityDBID, entity.AccountID)
	if err != nil {
		tx.Rollback()
		return errors.Errorf("Failed to update entity account_id: %s", err)
	}
	if err := checkRowCount("deleting old external_entity_id", res, 1); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return errors.Errorf("Failed to commit directory transaction: %s", err)
	}

	return nil
}

func checkRowCount(op string, res sql.Result, expected int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return errors.Errorf("Failed to get affect rows count: %s", err)
	}
	if n != expected {
		return errors.Errorf("Expected %d row when %s, got %d\n", expected, op, n)
	}
	return nil
}
