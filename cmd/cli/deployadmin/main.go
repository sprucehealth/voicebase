package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/cmd"
	"github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"
	"github.com/sprucehealth/backend/libs/golog"
)

const configPath = "~/.deploy.conf"

func loadConfig(app *boot.App) *config.Config {
	path, err := interpolatePath(configPath)
	if err != nil {
		golog.Fatalf("Invalid config path %s: %s", configPath, err)
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &config.Config{
				App: app,
			}
		}
		golog.Fatalf("Failed to read %s: %T", path, err)
	}
	var c config.Config
	if err := json.Unmarshal(b, &c); err != nil {
		golog.Fatalf("Failed to parse %s: %s", path, err)
	}
	c.App = app
	return &c
}

var commands = map[string]cmd.CommandNew{
	"clone_ecs_task_definition_to_deployable_config": cmd.NewCloneECSTaskDefinitionToDeployableConfigCmdCmd,
	"create_deployable":                              cmd.NewCreateDeployableCmd,
	"create_deployable_config":                       cmd.NewCreateDeployableConfigCmd,
	"create_deployable_group":                        cmd.NewCreateDeployableGroupCmd,
	"create_deployable_vector":                       cmd.NewCreateDeployableVectorCmd,
	"create_environment":                             cmd.NewCreateEnvironmentCmd,
	"create_environment_config":                      cmd.NewCreateEnvironmentConfigCmd,
	"list_deployables":                               cmd.NewListDeployablesCmd,
	"list_deployable_groups":                         cmd.NewListDeployableGroupsCmd,
	"list_deployable_configs":                        cmd.NewListDeployableConfigsCmd,
	"list_deployable_vectors":                        cmd.NewListDeployableVectorsCmd,
	"list_deployments":                               cmd.NewListDeploymentsCmd,
	"list_environments":                              cmd.NewListEnvironmentsCmd,
	"list_environment_configs":                       cmd.NewListEnvironmentConfigsCmd,
	"promote":                                        cmd.NewPromoteCmd,
	"report_build_complete":                          cmd.NewReportBuildCompleteCmd,
}

func main() {
	golog.Default().SetLevel(golog.INFO)
	cnf := loadConfig(boot.NewApp())
	flag.StringVar(&cnf.DeployAddr, "deploy_service", cnf.DeployAddr, "`host:port` of deploy service")
	flag.Parse()

	cmd := flag.Arg(0)

	for name, cfn := range commands {
		if name == cmd {
			c, err := cfn(cnf)
			if err != nil {
				golog.Fatalf(err.Error())
			}
			if err := c.Run(flag.Args()[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "FAILED: %s\n", err)
				os.Exit(2)
			}
			os.Exit(0)
		}
	}

	if cmd != "" {
		fmt.Printf("Unknown command '%s'\n", cmd)
	}

	fmt.Printf("Available commands:\n")
	for name := range commands {
		fmt.Printf("\t%s\n", name)
	}
	os.Exit(1)
}

func interpolatePath(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if p[0] == '~' {
		p = os.Getenv("HOME") + p[1:]
	}
	return filepath.Abs(p)
}
