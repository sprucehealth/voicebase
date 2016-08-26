package main

import (
	"context"
	"encoding/csv"
	"flag"
	"io"
	"os"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// updateVerifiedEmailCmd marks the email contact for patient entities
// identitied by the provied list of emails as verified.
type updateVerifiedEmailCmd struct {
	cnf          *config
	directoryCli directory.DirectoryClient
}

func newUpdateVerifiedEmailCmd(cnf *config) (command, error) {
	directoryCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	return &updateVerifiedEmailCmd{
		cnf:          cnf,
		directoryCli: directoryCli,
	}, nil
}

func (c *updateVerifiedEmailCmd) run(args []string) error {
	fs := flag.NewFlagSet("updateVerifiedEmail", flag.ExitOnError)
	emailFile := fs.String("email_file", "", "file containing email addresses to mark as verified for entities")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	if *emailFile == "" {
		return errors.New("email_file required")
	}

	emailFileReader, err := os.Open(*emailFile)
	if err != nil {
		return err
	}

	var emailAddresses []string
	r := csv.NewReader(emailFileReader)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		emailAddresses = append(emailAddresses, row[0])
	}

	for _, emailAddress := range emailAddresses {
		res, err := c.directoryCli.LookupEntitiesByContact(context.Background(), &directory.LookupEntitiesByContactRequest{
			ContactValue: emailAddress,
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			},
			RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
		})
		if err != nil && grpc.Code(err) != codes.NotFound {
			return errors.Errorf("Unable to lookup entity by contact for %s: %s", emailAddress, err)
		} else if res == nil {
			golog.Warningf("entity for contact %s not found. Skipping...", emailAddress)
			continue
		}

		for _, entity := range res.Entities {
			if entity.AccountID == "" {
				golog.Warningf("Entity %s has not created account yet. Skipping...", entity.ID)
				continue
			}

			for _, contact := range entity.Contacts {
				if contact.Value == emailAddress {
					contact.Verified = true
					golog.Infof("Updating entity contact %s to be marked as verified", contact.ID)
					break
				}
			}

			if _, err := c.directoryCli.UpdateContacts(context.Background(), &directory.UpdateContactsRequest{
				EntityID: entity.ID,
				Contacts: entity.Contacts,
			}); err != nil {
				return errors.Errorf("Unable to update contact for entity %s: %s", entity.ID, err)
			}
			golog.Infof("Updated contact for entity %s", entity.ID)
		}
	}
	return nil
}
