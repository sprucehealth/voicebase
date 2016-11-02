package cmd

import (
	"context"
	"flag"
	"time"

	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"
	"github.com/sprucehealth/backend/svc/deploy"
)

type listDeployableGroupsCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewListDeployableGroupsCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &listDeployableGroupsCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *listDeployableGroupsCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_deployable_groups", flag.ExitOnError)
	args = fs.Args()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.DeployableGroups(ctx, &deploy.DeployableGroupsRequest{})
	if err != nil {
		return err
	}

	printDeployableGroups(res.DeployableGroups)
	return nil
}
