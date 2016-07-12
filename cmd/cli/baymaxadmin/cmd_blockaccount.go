package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"context"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
)

type blockAccountCmd struct {
	cnf     *config
	dirCli  directory.DirectoryClient
	authCli auth.AuthClient
	dirDB   *sql.DB
}

func newBlockAccountCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	authCli, err := cnf.authClient()
	if err != nil {
		return nil, err
	}

	dirDB, err := cnf.directoryDB()
	if err != nil {
		return nil, err
	}

	return &blockAccountCmd{
		cnf:     cnf,
		dirCli:  dirCli,
		authCli: authCli,
		dirDB:   dirDB,
	}, nil
}

func (c *blockAccountCmd) run(args []string) error {
	fs := flag.NewFlagSet("blockaccount", flag.ExitOnError)
	accountID := fs.String("account_id", "", "accountID of the account to block")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	ctx := context.Background()

	scn := bufio.NewScanner(os.Stdin)

	if *accountID == "" {
		*accountID = prompt(scn, "Account ID: ")
	}
	if *accountID == "" {
		return errors.New("Account ID required")
	}

	entity, err := directory.SingleEntity(ctx, c.dirCli, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: *accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	})
	if err != nil {
		return err
	} else if entity.Type != directory.EntityType_INTERNAL {
		return fmt.Errorf("Expected original entity to be %s, got %s", directory.EntityType_INTERNAL, entity.Status)
	}

	fmt.Printf("Blocking account for entity:\n\n")
	entity, err = lookupAndDisplayEntity(ctx, c.dirCli, entity.ID, []directory.EntityInformation{
		directory.EntityInformation_CONTACTS,
		directory.EntityInformation_MEMBERSHIPS,
		directory.EntityInformation_EXTERNAL_IDS,
	})
	if err != nil {
		return errors.Trace(err)
	}
	if entity.Type != directory.EntityType_INTERNAL {
		return fmt.Errorf("Can only delete account for %s entities, not %s", directory.EntityType_INTERNAL, entity.Type)
	}
	if entity.Status != directory.EntityStatus_ACTIVE {
		return fmt.Errorf("Expected original entity to be %s, got %s", directory.EntityStatus_ACTIVE, entity.Status)
	}

	fmt.Println()
	if strings.ToLower(prompt(scn, "Block account entity [y/N]? ")) != "y" {
		return nil
	}

	// delete entity (harmless because it only updates status)
	if _, err := c.dirCli.DeleteEntity(ctx, &directory.DeleteEntityRequest{
		EntityID: entity.ID,
	}); err != nil {
		return errors.Trace(err)
	}

	// block account
	if _, err := c.authCli.BlockAccount(ctx, &auth.BlockAccountRequest{
		AccountID: *accountID,
	}); err != nil {
		return errors.Trace(err)
	}

	// because graphQL layer does not filter out members of org that are not ACTIVE, delete the membership of the
	// entity to its org
	// this will prevent any of its members from seeing the entity that was just blocked
	// TODO: Implement this through the directory service

	entityDBID, err := decodeID(entity.ID)
	if err != nil {
		return errors.Trace(err)
	}

	tx, err := c.dirDB.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	res, err := tx.Exec(`DELETE FROM entity_membership WHERE entity_id = ?`, entityDBID)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	if err := checkRowCount("deleting entity membership for the entity whose account is being blocked", res, 1); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Failed to commit directory transaction: %s", err)
	}

	return nil
}
