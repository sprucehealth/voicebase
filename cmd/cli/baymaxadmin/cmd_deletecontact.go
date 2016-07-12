package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"context"

	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type deleteContactCmd struct {
	cnf    *config
	dirCli directory.DirectoryClient
	excCli excomms.ExCommsClient
}

func newDeleteContactCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	excCli, err := cnf.exCommsClient()
	if err != nil {
		return nil, err
	}
	return &deleteContactCmd{
		cnf:    cnf,
		dirCli: dirCli,
		excCli: excCli,
	}, nil
}

func (c *deleteContactCmd) run(args []string) error {
	fs := flag.NewFlagSet("deletecontact", flag.ExitOnError)
	entityID := fs.String("entity_id", "", "ID of the entity that has the contact")
	contactID := fs.String("contact_id", "", "ID of the contact to delete")
	reason := fs.String("reason", "", "Reason for deprovisioning")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	ctx := context.Background()

	scn := bufio.NewScanner(os.Stdin)

	if *entityID == "" {
		*entityID = prompt(scn, "Entity ID: ")
	}
	if *entityID == "" {
		return errors.New("Entity ID required")
	}

	req := &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: *entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	}
	res, err := c.dirCli.LookupEntities(ctx, req)
	if err != nil && grpc.Code(err) != codes.NotFound {
		return err
	}

	switch len(res.Entities) {
	case 0:
		return errors.New("Entity not found")
	case 1:
	default:
		fmt.Fprintf(os.Stderr, "Expected 1 entity, got %d\n", len(res.Entities))
	}
	ent := res.Entities[0]

	displayEntity("", ent)

	if len(ent.Contacts) == 0 {
		return errors.New("Entity has no contacts")
	}

	if *contactID == "" {
		*contactID = prompt(scn, "Contact ID: ")
	}
	if *contactID == "" {
		return errors.New("Contact ID required")
	}

	// Make sure contact exists for the entity
	var contact *directory.Contact
	for _, c := range ent.Contacts {
		if c.ID == *contactID {
			contact = c
			break
		}
	}
	if contact == nil {
		return fmt.Errorf("Entity does not have contact %s", *contactID)
	}

	fmt.Printf("Contact to delete: %s type=%s value=%s provisioned=%v\n", contact.ID, contact.ContactType, contact.Value, contact.Provisioned)
	if strings.ToLower(prompt(scn, "Delete contact [y/N]? ")) != "y" {
		return nil
	}

	_, err = c.dirCli.DeleteContacts(ctx, &directory.DeleteContactsRequest{
		EntityID:         ent.ID,
		EntityContactIDs: []string{contact.ID},
	})
	if err != nil {
		return fmt.Errorf("Failed to delete contact: %s", err)
	}

	if contact.Provisioned && contact.ContactType == directory.ContactType_PHONE {
		switch strings.ToLower(prompt(scn, "Deprovision number [Y/n]? ")) {
		case "", "y":
		default:
			return nil
		}
		if *reason == "" {
			*reason = prompt(scn, "Reason: ")
		}
		_, err = c.excCli.DeprovisionPhoneNumber(ctx, &excomms.DeprovisionPhoneNumberRequest{
			PhoneNumber: contact.Value,
			Reason:      *reason,
		})
		if err != nil {
			return fmt.Errorf("Failed to deprovision number: %s", err)
		}
	}

	return nil
}
