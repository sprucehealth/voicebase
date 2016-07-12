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

type listDeployableVectorsCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewListDeployableVectorsCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &listDeployableVectorsCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *listDeployableVectorsCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_deployable_vectors", flag.ExitOnError)
	depID := fs.String("deployable_id", "", "The deployable to list vectors for")
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

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.DeployableVectors(ctx, &deploy.DeployableVectorsRequest{
		DeployableID: *depID,
	})
	if err != nil {
		return err
	}

	printDeployableVectors(res.Vectors)
	return nil
}
