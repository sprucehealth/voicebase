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

type createDeployableConfigCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewCreateDeployableConfigCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &createDeployableConfigCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *createDeployableConfigCmd) Run(args []string) error {
	fs := flag.NewFlagSet("create_deployable_config", flag.ExitOnError)
	depID := fs.String("deployable_id", "", "The deployable this config is for")
	envID := fs.String("environment_id", "", "The environment this config is for")
	sourceConfigID := fs.String("source_config_id", "", "The deployable config to copy for the original data")
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
	if *envID == "" {
		*envID = prompt(scn, "EnvironmentID: ")
		if *envID == "" {
			return errors.New("Environment ID is required")
		}
	}
	if *sourceConfigID == "" {
		*sourceConfigID = prompt(scn, "Source Config ID: ")
	}

	// TODO: Add the ability to import this via config
	var omitValues []string
	if *sourceConfigID != "" {
		for true {
			n := prompt(scn, "Omit Source Config Name: ")
			if n == "" {
				break
			}
			omitValues = append(omitValues, n)
		}
	}

	// TODO: Add the ability to import this via config
	configMap := make(map[string]string)
	for true {
		k := prompt(scn, "New Config Name: ")
		if k == "" {
			break
		}
		v := prompt(scn, "New Config Value: ")
		configMap[k] = v
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.CreateDeployableConfig(ctx, &deploy.CreateDeployableConfigRequest{
		DeployableID:   *depID,
		EnvironmentID:  *envID,
		SourceConfigID: *sourceConfigID,
		OmitFromSource: omitValues,
		Values:         configMap,
	})
	if err != nil {
		return err
	}

	printDeployableConfig(res.Config)
	return nil
}
