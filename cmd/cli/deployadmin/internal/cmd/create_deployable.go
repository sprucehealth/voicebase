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

type createDeployableCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewCreateDeployableCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &createDeployableCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *createDeployableCmd) Run(args []string) error {
	fs := flag.NewFlagSet("create_deployable", flag.ExitOnError)
	name := fs.String("name", "", "Name of the deployable")
	description := fs.String("description", "", "Description of the group")
	groupID := fs.String("deployable_group_id", "", "The deployable group that this deployable is for")
	gitURL := fs.String("git_url", "https://github.com/SpruceHealth/backend", "The URL of the git repo for this deployable Default: 'https://github.com/SpruceHealth/backend'")
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

	res, err := c.deployCli.CreateDeployable(ctx, &deploy.CreateDeployableRequest{
		Name:              *name,
		Description:       *description,
		DeployableGroupID: *groupID,
		GitURL:            *gitURL,
	})
	if err != nil {
		return err
	}

	printDeployable(res.Deployable)
	return nil
}
