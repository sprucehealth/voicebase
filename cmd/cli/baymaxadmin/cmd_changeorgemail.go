package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type changeOrgEmailCmd struct {
	cnf    *config
	dirCli directory.DirectoryClient
	excCli excomms.ExCommsClient
}

var domainsPerEnvironment = map[string]string{
	"prod":    "sprucecare",
	"staging": "amdava",
	"dev":     "vimdrop",
}

func newChangeOrgEmailCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	excCli, err := cnf.exCommsClient()
	if err != nil {
		return nil, err
	}

	if cnf.Env == "" {
		return nil, errors.New("cannot instantiate command when environment not specified")
	}

	return &changeOrgEmailCmd{
		cnf:    cnf,
		dirCli: dirCli,
		excCli: excCli,
	}, nil
}

func (c *changeOrgEmailCmd) run(args []string) error {
	fs := flag.NewFlagSet("changeorgemail", flag.ExitOnError)
	orgEntityID := fs.String("org_entity_id", "", "ID of the org entity for whom to change the email")
	domain := fs.String("domain", "", "Subdomain for the email address. That is, the y in x@y.sprucecare.com")
	localPart := fs.String("local_part", "", "Localpart of the email address. That is, the x in x@y.sprucecare.com")
	ctx := context.Background()

	if err := fs.Parse(args); err != nil {
		return err
	}

	scn := bufio.NewScanner(os.Stdin)
	if *orgEntityID == "" {
		*orgEntityID = prompt(scn, "Entity ID: ")
	}
	if *orgEntityID == "" {
		return errors.New("Org Entity ID required")
	}

	if *domain == "" {
		*domain = prompt(scn, "Domain: ")
	}
	if *domain == "" {
		return errors.New("Domain required")
	}

	if *localPart == "" {
		*localPart = prompt(scn, "Local part: ")
	}
	if *localPart == "" {
		return errors.New("Local part required")
	}

	newEmailAddress := strings.ToLower(fmt.Sprintf("%s@%s.%s.com", *localPart, *domain, domainsPerEnvironment[c.cnf.Env]))

	// ensure that no one already has the email address that the provider wants
	entityFound, err := directory.SingleEntityByContact(ctx, c.dirCli, &directory.LookupEntitiesByContactRequest{
		ContactValue: newEmailAddress,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	if err != nil && errors.Cause(err) != directory.ErrEntityNotFound {
		return err
	} else if entityFound != nil {
		if entityFound.ID != *orgEntityID {
			return errors.New("email address already taken")
		}
		return nil
	}

	// only allow updating of email address and entity if the entityID = org and there are no other provisioned
	// pieces of contact at the org or member level
	orgEntity, err := directory.SingleEntity(ctx, c.dirCli, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: *orgEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_CONTACTS,
			},
		},
	})
	if err != nil {
		return errors.New("Cannot retrieve entity: " + err.Error())
	} else if orgEntity.Type != directory.EntityType_ORGANIZATION {
		return errors.New("Can only change email address for org")
	}

	// ensure that there is just a single email provisioned at the org level
	var contactToModify *directory.Contact
	for _, contact := range orgEntity.Contacts {
		if contact.ContactType == directory.ContactType_EMAIL && contact.Provisioned {
			if contactToModify != nil {
				return errors.New("Org has more than 1 Spruce email address provisioned")
			}
			contactToModify = contact
		}
	}

	if contactToModify == nil {
		return errors.New("Org does not have a spruce email address")
	}

	// ensure that no members of the org have a provisioned email address
	// the reason we don't want to update the email address in this case is because modifying an entity
	// domain has a global impact on all email addresses pertaining to the org
	for _, m := range orgEntity.Members {
		for _, contact := range m.Contacts {
			if contact.Provisioned && contact.ContactType == directory.ContactType_EMAIL {
				return errors.New("One of the org members has a provisioned email")
			}
		}
	}

	// check if domain already taken
	domainUnchanged := false
	res, err := c.dirCli.LookupEntityDomain(ctx, &directory.LookupEntityDomainRequest{
		Domain: *domain,
	})
	if err != nil && grpc.Code(err) != codes.NotFound {
		return errors.Trace(err)
	} else if res != nil {
		if *orgEntityID != res.EntityID {
			return errors.Trace(fmt.Errorf("domain %s already taken by %s", *domain, res.EntityID))
		}
		domainUnchanged = true
	}

	if !domainUnchanged {
		// Update the entity domain
		if _, err := c.dirCli.UpdateEntityDomain(ctx, &directory.UpdateEntityDomainRequest{
			EntityID: *orgEntityID,
			Domain:   *domain,
		}); err != nil {
			return errors.Trace(err)
		}
	}

	// delete the existing contact for the entity
	if _, err := c.dirCli.DeleteContacts(ctx, &directory.DeleteContactsRequest{
		EntityID:         *orgEntityID,
		EntityContactIDs: []string{contactToModify.ID},
	}); err != nil {
		return errors.Trace(err)
	}

	// deprovision the email address from the excomms layer
	if _, err := c.excCli.DeprovisionEmail(ctx, &excomms.DeprovisionEmailRequest{
		Email:  contactToModify.Value,
		Reason: "Support request change",
	}); err != nil {
		return errors.Trace(err)
	}

	// create new email address for the entity
	if _, err := c.dirCli.CreateContact(ctx, &directory.CreateContactRequest{
		EntityID: *orgEntityID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Provisioned: true,
			Value:       newEmailAddress,
		},
	}); err != nil {
		return errors.Trace(err)
	}

	// provision email at excomms layer
	if _, err := c.excCli.ProvisionEmailAddress(ctx, &excomms.ProvisionEmailAddressRequest{
		EmailAddress: newEmailAddress,
		ProvisionFor: *orgEntityID,
	}); err != nil {
		return errors.Trace(err)
	}

	// success!
	if _, err := lookupAndDisplayEntity(ctx, c.dirCli, *orgEntityID, []directory.EntityInformation{directory.EntityInformation_CONTACTS}); err != nil {
		return err
	}

	return nil
}
