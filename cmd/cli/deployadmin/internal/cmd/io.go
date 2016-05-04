package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sprucehealth/backend/svc/deploy"
)

func prompt(scn *bufio.Scanner, prompt string) string {
	fmt.Print(prompt)
	if !scn.Scan() {
		os.Exit(1)
	}
	return strings.TrimSpace(scn.Text())
}

func pprint(fs string, args ...interface{}) {
	fmt.Printf(fs, args...)
}

func printDeployableGroups(dgs []*deploy.DeployableGroup) {
	for _, dg := range dgs {
		printDeployableGroup(dg)
	}
}

func printDeployableGroup(dg *deploy.DeployableGroup) {
	pprint("Deployable Group: %s (name %s) (description %q)\n", dg.ID, dg.Name, dg.Description)
}

func printEnvironments(envs []*deploy.Environment) {
	for _, env := range envs {
		printEnvironment(env)
	}
}

func printEnvironment(env *deploy.Environment) {
	pprint("Environment: %s (name %s) (description %q) (deployable group %s) (prod %v)\n", env.ID, env.Name, env.Description, env.DeployableGroupID, env.IsProd)
}

func printDeployables(deps []*deploy.Deployable) {
	for _, dep := range deps {
		printDeployable(dep)
	}
}

func printDeployable(dep *deploy.Deployable) {
	pprint("Deployable: %s (name %s) (description %q) (deployable group %s) (git url %s)\n", dep.ID, dep.Name, dep.Description, dep.DeployableGroupID, dep.GitURL)
}

func printEnvironmentConfigs(cs []*deploy.EnvironmentConfig) {
	for _, c := range cs {
		printEnvironmentConfig(c)
	}
}

func printEnvironmentConfig(c *deploy.EnvironmentConfig) {
	pprint("Environment Config: %s (environment %s) (status %q)\n", c.ID, c.EnvironmentID, c.Status)
	for k, v := range c.Values {
		pprint("\tName: %s Value: %s\n", k, v)
	}
}

func printDeployableConfigs(cs []*deploy.DeployableConfig) {
	for _, c := range cs {
		printDeployableConfig(c)
	}
}

func printDeployableConfig(c *deploy.DeployableConfig) {
	pprint("Deployable Config: %s (deployable %s) (environment %s) (status %q)\n", c.ID, c.DeployableID, c.EnvironmentID, c.Status)
	for k, v := range c.Values {
		pprint("\tName: %s Value: %s\n", k, v)
	}
}

func printDeployableVectors(vs []*deploy.DeployableVector) {
	for _, v := range vs {
		printDeployableVector(v)
	}
}

func printDeployableVector(v *deploy.DeployableVector) {
	sournceEnvironment := "none"
	if v.SourceType == deploy.DeployableVector_ENVIRONMENT_ID {
		sournceEnvironment = v.GetEnvironmentID()
	}
	pprint("Deployable Vector: %s (deployable %s) (source type %s) (source environment %s) (target environment %s)\n", v.ID, v.DeployableID, v.SourceType.String(), sournceEnvironment, v.TargetEnvironmentID)
}

func printDeployments(ds []*deploy.Deployment) {
	for _, d := range ds {
		printDeployment(d)
	}
}

func printDeployment(d *deploy.Deployment) {
	pprint("Deployment: %s (deployable %s) (environment %s) (status %s) (deployable config %s) (deployable vector %s) (type %s) (build number %s) (deployment number %d) (git hash %s)\n",
		d.ID, d.DeployableID, d.EnvironmentID, d.Status.String(), d.DeployableConfigID, d.DeployableVectorID, d.Type.String(), d.BuildNumber, d.DeploymentNumber, d.GitHash)
	switch d.Type {
	case deploy.Deployment_ECS:
		ecsDeployment := d.GetEcsDeployment()
		pprint("\tECS Deployment: (image %s) (cluster config name %s)\n", ecsDeployment.Image, ecsDeployment.ClusterDeployableConfigName)
	default:
		pprint("\tUnknown Deployment Type: %+v", d.DeploymentOneof)
	}
}
