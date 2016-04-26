package cmd

import (
	"bufio"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"
	"github.com/sprucehealth/backend/svc/deploy"
	"golang.org/x/net/context"
)

type createDeployableGroupCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewCreateDeployableGroupCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &createDeployableGroupCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *createDeployableGroupCmd) Run(args []string) error {
	fs := flag.NewFlagSet("create_deployable_group", flag.ExitOnError)
	name := fs.String("name", "", "Name of the group")
	description := fs.String("description", "", "Description of the group")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *name == "" {
		*name = prompt(scn, "Name: ")
		if *name == "" {
			return errors.New("Group Name is required")
		}
	}
	if *description == "" {
		*description = prompt(scn, "Description: ")
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.CreateDeployableGroup(ctx, &deploy.CreateDeployableGroupRequest{
		Name:        *name,
		Description: *description,
	})
	if err != nil {
		return err
	}

	printDeployableGroup(res.DeployableGroup)
	return nil
}
