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

type createEnvironmentConfigCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewCreateEnvironmentConfigCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &createEnvironmentConfigCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *createEnvironmentConfigCmd) Run(args []string) error {
	fs := flag.NewFlagSet("create_environment_config", flag.ExitOnError)
	envID := fs.String("environment_id", "", "The environment this config is for")
	sourceConfigID := fs.String("source_config_id", "", "The environment config to copy for the original data")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

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

	res, err := c.deployCli.CreateEnvironmentConfig(ctx, &deploy.CreateEnvironmentConfigRequest{
		EnvironmentID:  *envID,
		SourceConfigID: *sourceConfigID,
		OmitFromSource: omitValues,
		Values:         configMap,
	})
	if err != nil {
		return err
	}

	printEnvironmentConfig(res.Config)
	return nil
}
