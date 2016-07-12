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

type reportBuildCompleteCmd struct {
	cnf       *config.Config
	deployCli deploy.DeployClient
}

func NewReportBuildCompleteCmd(cnf *config.Config) (Command, error) {
	deployCli, err := cnf.DeployClient()
	if err != nil {
		return nil, err
	}
	return &reportBuildCompleteCmd{
		cnf:       cnf,
		deployCli: deployCli,
	}, nil
}

// TODO: Make this more flexible rather than just docker builds
func (c *reportBuildCompleteCmd) Run(args []string) error {
	fs := flag.NewFlagSet("report_build_complete", flag.ExitOnError)
	depID := fs.String("deployable_id", "", "The deployable this build is for")
	buildNumber := fs.String("build_number", "", "The build number for this build artifact")
	gitHash := fs.String("git_hash", "", "The git has of this build")
	image := fs.String("image", "", "The docker image for this artifact")
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
	if *buildNumber == "" {
		*buildNumber = prompt(scn, "Build Number: ")
		if *buildNumber == "" {
			return errors.New("Build Number is required")
		}
	}
	if *image == "" {
		*image = prompt(scn, "Image: ")
		if *image == "" {
			return errors.New("Image is required")
		}
	}

	if *gitHash == "" {
		*gitHash = "NONE_PROVIDED"
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	res, err := c.deployCli.ReportBuildComplete(ctx, &deploy.ReportBuildCompleteRequest{
		DeployableID: *depID,
		BuildNumber:  *buildNumber,
		ArtifactType: deploy.ReportBuildCompleteRequest_DOCKER_IMAGE,
		BuildArtifactOneof: &deploy.ReportBuildCompleteRequest_DockerImage{
			DockerImage: *image,
		},
		GitHash: *gitHash,
	})
	if err != nil {
		return err
	}

	printDeployments(res.Deployments)
	return nil
}
