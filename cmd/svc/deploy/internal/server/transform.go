package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/dal"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/deploy"
)

func transformDeployableGroupsToResponse(dgs []*dal.DeployableGroup) []*deploy.DeployableGroup {
	rs := make([]*deploy.DeployableGroup, len(dgs))
	for i, dg := range dgs {
		rs[i] = transformDeployableGroupToResponse(dg)
	}
	return rs
}

func transformDeployableGroupToResponse(dg *dal.DeployableGroup) *deploy.DeployableGroup {
	return &deploy.DeployableGroup{
		ID:                dg.ID.String(),
		Name:              dg.Name,
		Description:       dg.Description,
		CreatedTimestamp:  uint64(dg.Created.Unix()),
		ModifiedTimestamp: uint64(dg.Modified.Unix()),
	}
}

func transformEnvironmentsToResponse(envs []*dal.Environment) []*deploy.Environment {
	rs := make([]*deploy.Environment, len(envs))
	for i, env := range envs {
		rs[i] = transformEnvironmentToResponse(env)
	}
	return rs
}

func transformEnvironmentToResponse(env *dal.Environment) *deploy.Environment {
	return &deploy.Environment{
		ID:                env.ID.String(),
		Name:              env.Name,
		Description:       env.Description,
		IsProd:            env.IsProd,
		DeployableGroupID: env.DeployableGroupID.String(),
		CreatedTimestamp:  uint64(env.Created.Unix()),
		ModifiedTimestamp: uint64(env.Modified.Unix()),
	}
}

func transformDeployablesToResponse(deps []*dal.Deployable) []*deploy.Deployable {
	rs := make([]*deploy.Deployable, len(deps))
	for i, dep := range deps {
		rs[i] = transformDeployableToResponse(dep)
	}
	return rs
}

func transformDeployableToResponse(dep *dal.Deployable) *deploy.Deployable {
	return &deploy.Deployable{
		ID:                dep.ID.String(),
		Name:              dep.Name,
		Description:       dep.Description,
		DeployableGroupID: dep.DeployableGroupID.String(),
		CreatedTimestamp:  uint64(dep.Created.Unix()),
		ModifiedTimestamp: uint64(dep.Modified.Unix()),
	}
}

func transformEnvironmentConfigToResponse(config *dal.EnvironmentConfig, values map[string]string) *deploy.EnvironmentConfig {
	return &deploy.EnvironmentConfig{
		ID:            config.ID.String(),
		EnvironmentID: config.EnvironmentID.String(),
		Status:        config.Status.String(),
		Values:        values,
	}
}

func transformDeployableConfigToResponse(config *dal.DeployableConfig, values map[string]string) *deploy.DeployableConfig {
	return &deploy.DeployableConfig{
		ID:            config.ID.String(),
		DeployableID:  config.DeployableID.String(),
		EnvironmentID: config.EnvironmentID.String(),
		Status:        config.Status.String(),
		Values:        values,
	}
}

func transformDeployableVectorsToResponse(vs []*dal.DeployableVector) []*deploy.DeployableVector {
	rs := make([]*deploy.DeployableVector, len(vs))
	for i, v := range vs {
		rs[i] = transformDeployableVectorToResponse(v)
	}
	return rs
}

func transformDeployableVectorToResponse(dv *dal.DeployableVector) *deploy.DeployableVector {
	rdv := &deploy.DeployableVector{
		ID:                  dv.ID.String(),
		DeployableID:        dv.DeployableID.String(),
		TargetEnvironmentID: dv.TargetEnvironmentID.String(),
	}
	switch dv.SourceType {
	case dal.DeployableVectorSourceTypeBuild:
		rdv.SourceType = deploy.DeployableVector_BUILD
	case dal.DeployableVectorSourceTypeEnvironmentID:
		rdv.SourceType = deploy.DeployableVector_ENVIRONMENT_ID
		rdv.DeploymentSourceOneof = &deploy.DeployableVector_EnvironmentID{
			EnvironmentID: dv.SourceEnvironmentID.String(),
		}
	default:
		golog.Errorf("Unknown source type for %v", dv)
	}
	return rdv
}

func transformDeploymentsToResponse(ds []*dal.Deployment) ([]*deploy.Deployment, error) {
	var err error
	rs := make([]*deploy.Deployment, len(ds))
	for i, d := range ds {
		rs[i], err = transformDeploymentToResponse(d)
		if err != nil {
			return nil, err
		}
	}
	return rs, nil
}

func transformDeploymentToResponse(d *dal.Deployment) (*deploy.Deployment, error) {
	rd := &deploy.Deployment{
		ID:                 d.ID.String(),
		DeploymentNumber:   d.DeploymentNumber,
		DeployableID:       d.DeployableID.String(),
		EnvironmentID:      d.EnvironmentID.String(),
		DeployableConfigID: d.DeployableConfigID.String(),
		DeployableVectorID: d.DeployableVectorID.String(),
		BuildNumber:        d.BuildNumber,
	}
	switch d.Type {
	case dal.DeploymentTypeEcs:
		rd.Type = deploy.Deployment_ECS
		ecsDeployment := &deploy.ECSDeployment{}
		if err := ecsDeployment.Unmarshal(d.Data); err != nil {
			return nil, err
		}
		rd.DeploymentOneof = &deploy.Deployment_EcsDeployment{
			EcsDeployment: ecsDeployment,
		}
	default:
		return nil, fmt.Errorf("Unhandled deployment type for %v", d)
	}

	switch d.Status {
	case dal.DeploymentStatusPending:
		rd.Status = deploy.Deployment_PENDING
	case dal.DeploymentStatusInProgress:
		rd.Status = deploy.Deployment_IN_PROGRESS
	case dal.DeploymentStatusComplete:
		rd.Status = deploy.Deployment_COMPLETE
	case dal.DeploymentStatusFailed:
		rd.Status = deploy.Deployment_FAILED
	default:
		return nil, fmt.Errorf("Unhandled deployment status for %v", d)
	}
	return rd, nil
}
