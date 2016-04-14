package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type accountCmd struct {
	cnf     *config
	authCli auth.AuthClient
	dirCli  directory.DirectoryClient
}

func newAccountCmd(cnf *config) (command, error) {
	authCli, err := cnf.authClient()
	if err != nil {
		return nil, err
	}
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	return &accountCmd{
		cnf:     cnf,
		authCli: authCli,
		dirCli:  dirCli,
	}, nil
}

func (c *accountCmd) run(args []string) error {
	fs := flag.NewFlagSet("account", flag.ExitOnError)
	accountID := fs.String("account_id", "", "ID of the account")
	email := fs.String("email", "", "Email of the account")
	withEntities := fs.Bool("entities", false, "Display related entities")
	withMemberships := fs.Bool("memberships", false, "Display memberships for related entities")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *accountID == "" {
		*accountID = prompt(scn, "Account ID: ")
	}
	if *accountID == "" {
		if *email == "" {
			*email = prompt(scn, "Email: ")
		}
		if *email == "" {
			return errors.New("Account ID or email is required")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.authCli.GetAccount(ctx, &auth.GetAccountRequest{
		AccountID:    *accountID,
		AccountEmail: *email,
	})
	if grpc.Code(err) == codes.NotFound {
		return errors.New("Account not found")
	} else if err != nil {
		return err
	}

	fmt.Printf("Account %s (type %s) (firstName %s) (lastName %s)\n", res.Account.ID, res.Account.Type, res.Account.FirstName, res.Account.LastName)

	if *withEntities {
		req := &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: *accountID,
			},
			RequestedInformation: &directory.RequestedInformation{},
		}
		if *withMemberships {
			req.RequestedInformation.EntityInformation = append(req.RequestedInformation.EntityInformation, directory.EntityInformation_MEMBERSHIPS)
		}
		res, err := c.dirCli.LookupEntities(ctx, req)
		if err != nil && grpc.Code(err) != codes.NotFound {
			return err
		}
		if len(res.Entities) != 0 {
			w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
			fmt.Println("Entities:")
			fmt.Fprintf(w, "    ID\tDisplay Name\tType\tStatus\n")
			hasMemberships := false
			for _, e := range res.Entities {
				fmt.Fprintf(w, "    %s\t%s\t%s\t%s\n", e.ID, e.Info.DisplayName, e.Type, e.Status)
				if len(e.Memberships) != 0 {
					hasMemberships = true
				}
			}
			if err := w.Flush(); err != nil {
				golog.Fatalf(err.Error())
			}

			if hasMemberships {
				fmt.Println("Memberships:")
				fmt.Fprintf(w, "    ID\tDisplay Name\tType\tStatus\n")
				for _, e := range res.Entities {
					for _, em := range e.Memberships {
						fmt.Fprintf(w, "    %s\t%s\t%s\t%s\n", em.ID, em.Info.DisplayName, em.Type, em.Status)
					}
				}
				if err := w.Flush(); err != nil {
					golog.Fatalf(err.Error())
				}
			}
		}
	}

	return nil
}
