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

type promoteGroupCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewPromoteGroupCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &promoteGroupCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *promoteGroupCmd) Run(args []string) error {
	fs := flag.NewFlagSet("promote_group", flag.ExitOnError)
	depGroupID := fs.String("deployable_group", "", "The deployable group to promote")
	buildNumber := fs.String("build_number", "", "The deployable group build number promote")
	environmentID := fs.String("environmentID", "", "The environment from which to promote")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *depGroupID == "" {
		*depGroupID = prompt(scn, "Deployable Group ID: ")
		if *depGroupID == "" {
			return errors.New("Deployable Group ID is required")
		}
	}

	if *buildNumber == "" {
		*buildNumber = prompt(scn, "Build Number: ")
		if *buildNumber == "" {
			return errors.New("Build Number is required")
		}
	}

	if *environmentID == "" {
		*environmentID = prompt(scn, "EnvironmentID: ")
		if *environmentID == "" {
			return errors.New("Environment ID is required")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.PromoteGroup(ctx, &deploy.PromoteGroupRequest{
		DeployableGroupID: *depGroupID,
		BuildNumber:       *buildNumber,
		EnvironmentID:     *environmentID,
	})
	if err != nil {
		return err
	}

	printDeployments(res.Deployments)
	return nil
}
