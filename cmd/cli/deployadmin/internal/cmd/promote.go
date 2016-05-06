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

type promoteCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewPromoteCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &promoteCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *promoteCmd) Run(args []string) error {
	fs := flag.NewFlagSet("promoteCmd", flag.ExitOnError)
	depID := fs.String("deployment_id", "", "The deployment to promote")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *depID == "" {
		*depID = prompt(scn, "DeploymentID: ")
		if *depID == "" {
			return errors.New("Deployment ID is required")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.Promote(ctx, &deploy.PromotionRequest{
		DeploymentID: *depID,
	})
	if err != nil {
		return err
	}

	printDeployments(res.Deployments)
	return nil
}
