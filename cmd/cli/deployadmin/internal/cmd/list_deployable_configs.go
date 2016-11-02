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

type createListDeployableConfigsCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewListDeployableConfigsCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &createListDeployableConfigsCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *createListDeployableConfigsCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_deployable_configs", flag.ExitOnError)
	depID := fs.String("deployable_id", "", "The deployable for which we should list configs")
	envID := fs.String("environment_id", "", "The environment for which we should list configs")
	status := fs.String("status", "ACTIVE", "The environment config status to filter on")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *depID == "" {
		*depID = prompt(scn, "Deployable ID: ")
		if *depID == "" {
			return errors.New("Deployable ID is required")
		}
	}
	if *envID == "" {
		*envID = prompt(scn, "Environment ID: ")
		if *envID == "" {
			return errors.New("Environment ID is required")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.DeployableConfigs(ctx, &deploy.DeployableConfigsRequest{
		DeployableID:  *depID,
		EnvironmentID: *envID,
		Status:        *status,
	})
	if err != nil {
		return err
	}

	printDeployableConfigs(res.Configs)
	return nil
}
