package cmd

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"
	"github.com/sprucehealth/backend/svc/deploy"
	"golang.org/x/net/context"
)

type createDeployableVectorCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewCreateDeployableVectorCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &createDeployableVectorCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

func (c *createDeployableVectorCmd) Run(args []string) error {
	fs := flag.NewFlagSet("create_deployable_vector", flag.ExitOnError)
	depID := fs.String("deployable_id", "", "The deployable that this vector is for")
	sourceType := fs.String("source_type", "", "The source type for this vector")
	sourceValue := fs.String("source_value", "", "The source value for this vector")
	targetEnvID := fs.String("target_environment_id", "", "The  vector")
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
	if *sourceType == "" {
		*sourceType = prompt(scn, "SourceType: ")
		if *sourceType == "" {
			return errors.New("Source Type is required")
		}
	}
	iSourceType, ok := deploy.CreateDeployableVectorRequest_DeployableVectorSourceType_value[strings.ToUpper(*sourceType)]
	if !ok {
		return fmt.Errorf("Source Type is %s is not valid", *sourceType)
	}
	dSourceType := deploy.CreateDeployableVectorRequest_DeployableVectorSourceType(iSourceType)

	if dSourceType == deploy.CreateDeployableVectorRequest_ENVIRONMENT_ID {
		if *sourceValue == "" {
			*sourceValue = prompt(scn, "Source EnvironmentID: ")
			if *sourceValue == "" {
				return errors.New("Source Environment ID is required")
			}
		}
	}
	if *targetEnvID == "" {
		*targetEnvID = prompt(scn, "Target EnvironmentID: ")
		if *targetEnvID == "" {
			return errors.New("Target Environment ID is required")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	req := &deploy.CreateDeployableVectorRequest{
		DeployableID:        *depID,
		SourceType:          dSourceType,
		TargetEnvironmentID: *targetEnvID,
	}

	if dSourceType == deploy.CreateDeployableVectorRequest_ENVIRONMENT_ID {
		req.DeploymentSourceOneof = &deploy.CreateDeployableVectorRequest_SourceEnvironmentID{
			SourceEnvironmentID: *sourceValue,
		}
	}

	res, err := c.deployCli.CreateDeployableVector(ctx, req)
	if err != nil {
		return err
	}

	printDeployableVector(res.Vector)
	return nil
}
