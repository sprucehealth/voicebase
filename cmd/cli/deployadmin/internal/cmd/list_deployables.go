package cmd

import (
	"bufio"
	"errors"
	"flag"
	"os"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"
	"github.com/sprucehealth/backend/svc/deploy"
)

type listDeployablesCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewListDeployablesCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &listDeployablesCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *listDeployablesCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_deployables", flag.ExitOnError)
	groupID := fs.String("deployable_group_id", "", "The deployable group to list deployables for")
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

	res, err := c.deployCli.Deployables(ctx, &deploy.DeployablesRequest{
		DeployableGroupID: *groupID,
	})
	if err != nil {
		return err
	}

	printDeployables(res.Deployables)
	return nil
}
