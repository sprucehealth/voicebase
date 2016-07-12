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

type createEnvironmentCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewCreateEnvironmentCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &createEnvironmentCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *createEnvironmentCmd) Run(args []string) error {
	fs := flag.NewFlagSet("create_environment", flag.ExitOnError)
	name := fs.String("name", "", "Name of the group")
	description := fs.String("description", "", "Description of the group")
	groupID := fs.String("deployable_group_id", "", "The deployable group that this env is for")
	isProd := fs.Bool("is_prod", false, "A flag representing if this environment is for a prod environment")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *name == "" {
		*name = prompt(scn, "Name: ")
		if *name == "" {
			return errors.New("Name is required")
		}
	}
	if *description == "" {
		*description = prompt(scn, "Description: ")
	}
	if *groupID == "" {
		*groupID = prompt(scn, "Deployable Group ID: ")
		if *groupID == "" {
			return errors.New("Deployable Group ID is required")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.CreateEnvironment(ctx, &deploy.CreateEnvironmentRequest{
		Name:              *name,
		Description:       *description,
		IsProd:            *isProd,
		DeployableGroupID: *groupID,
	})
	if err != nil {
		return err
	}

	printEnvironment(res.Environment)
	return nil
}
