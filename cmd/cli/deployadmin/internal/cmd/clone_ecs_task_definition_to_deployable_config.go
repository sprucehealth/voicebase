package cmd

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/svc/deploy"
)

type cloneECSTaskDefinitionToDeployableConfigCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
	stsCli    stsiface.STSAPI
}

func NewCloneECSTaskDefinitionToDeployableConfigCmdCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	stsCli, err := cnf.STSClient()
	if err != nil {
		return nil, err
	}
	return &cloneECSTaskDefinitionToDeployableConfigCmd{
		cnf:       cnf,
		deployCli: deployCli,
		stsCli:    stsCli,
	}, nil
}

func (c *cloneECSTaskDefinitionToDeployableConfigCmd) Run(args []string) error {
	fs := flag.NewFlagSet("clone_ecs_task_definition_to_deployable_config", flag.ExitOnError)
	depID := fs.String("deployable_id", "", "The deployable this config is for")
	envID := fs.String("environment_id", "", "The environment this config is for")
	familyName := fs.String("task_definition_family", "", "The ecs task definition family to transform into deployable config")
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
	if *familyName == "" {
		*familyName = prompt(scn, "Family Name: ")
		if *envID == "" {
			return errors.New("Family Name is required")
		}
	}

	// Assume the correct role. For now hack this.
	// TODO: Figure out how to track roles vs envs
	aECSCli, err := awsutil.AssumedECSCli(c.stsCli, "arn:aws:iam::758505115169:role/dev-deploy-ecs", fmt.Sprintf("clone-%s-%s-%s", depID, envID, familyName))
	if err != nil {
		return err
	}

	res, err := aECSCli.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: familyName,
	})
	if err != nil {
		return err
	}
	configMap := make(map[string]string)
	for _, v := range res.TaskDefinition.ContainerDefinitions[0].Environment {
		configMap[*v.Name] = *v.Value
	}
	configMap[`ECS_CONFIG_CPU`] = strconv.FormatInt(*res.TaskDefinition.ContainerDefinitions[0].Cpu, 10)
	configMap[`ECS_CONFIG_MEMORY`] = strconv.FormatInt(*res.TaskDefinition.ContainerDefinitions[0].Memory, 10)
	for i, pm := range res.TaskDefinition.ContainerDefinitions[0].PortMappings {
		configMap[fmt.Sprintf("ECS_CONFIG_PORT_MAPPING_%d", i)] = fmt.Sprintf("%d:%d:%s", *pm.ContainerPort, *pm.HostPort, *pm.Protocol)
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	cRes, err := c.deployCli.CreateDeployableConfig(ctx, &deploy.CreateDeployableConfigRequest{
		DeployableID:  *depID,
		EnvironmentID: *envID,
		Values:        configMap,
	})
	if err != nil {
		return err
	}

	printDeployableConfig(cRes.Config)
	return nil
}
