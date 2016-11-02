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

type createListEnvironmentConfigsCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewListEnvironmentConfigsCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &createListEnvironmentConfigsCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *createListEnvironmentConfigsCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_environment_configs", flag.ExitOnError)
	envID := fs.String("environment_id", "", "The environment for which we should list configs")
	status := fs.String("status", "ACTIVE", "The environment config status to filter on")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *envID == "" {
		*envID = prompt(scn, "Environment ID: ")
		if *envID == "" {
			return errors.New("Environment ID is required")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.EnvironmentConfigs(ctx, &deploy.EnvironmentConfigsRequest{
		EnvironmentID: *envID,
		Status:        *status,
	})
	if err != nil {
		return err
	}

	printEnvironmentConfigs(res.Configs)
	return nil
}
