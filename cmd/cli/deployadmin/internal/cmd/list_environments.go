package cmd

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"
	"github.com/sprucehealth/backend/svc/deploy"
)

type listEnvironmentsCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewListEnvironmentsCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &listEnvironmentsCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *listEnvironmentsCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_environments", flag.ExitOnError)
	groupID := fs.String("deployable_group_id", "", "The deployable group to list envs for")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *groupID == "" {
		*groupID = prompt(scn, "Deployable Group ID: ")
		if *groupID == "" {
			return errors.New("Deployable Group ID is required")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.Environments(ctx, &deploy.EnvironmentsRequest{
		By: &deploy.EnvironmentsRequest_DeployableGroupID{
			DeployableGroupID: *groupID,
		},
	})
	if err != nil {
		return err
	}

	printEnvironments(res.Environments)
	return nil
}
