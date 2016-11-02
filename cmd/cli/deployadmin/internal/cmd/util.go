package cmd

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/svc/deploy"
)

func environmentNames(ctx context.Context, cli deploy.DeployClient, deployableGroupID string) (map[string]string, error) {
	res, err := cli.Environments(ctx, &deploy.EnvironmentsRequest{
		By: &deploy.EnvironmentsRequest_DeployableGroupID{
			DeployableGroupID: deployableGroupID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch environments: %s", err)
	}
	idToName := make(map[string]string, len(res.Environments))
	for _, e := range res.Environments {
		idToName[e.ID] = e.Name
	}
	return idToName, nil
}
