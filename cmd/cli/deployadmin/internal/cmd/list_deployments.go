package cmd

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"
	"github.com/sprucehealth/backend/svc/deploy"
)

type listDeploymentsCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewListDeploymentsCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &listDeploymentsCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *listDeploymentsCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_deployments", flag.ExitOnError)
	depID := fs.String("deployable_id", "", "The deployable to list deployments for")
	status := fs.String("status", "", "The status to filter to")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *depID == "" {
		*depID = prompt(scn, "DeployableID: ")
		if *depID == "" {
			return errors.New("Deployable ID is required")
		}
	}

	rStatus := deploy.DeploymentsRequest_ANY
	if *status != "" {
		sta, ok := deploy.DeploymentsRequest_Status_value[strings.ToUpper(*status)]
		if !ok {
			return fmt.Errorf("Deployment status %s is not valid", *status)
		}
		rStatus = deploy.DeploymentsRequest_Status(sta)
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.Deployments(ctx, &deploy.DeploymentsRequest{
		DeployableID: *depID,
		Status:       rStatus,
	})
	if err != nil {
		return err
	}

	printDeployments(res.Deployments)
	return nil
}
