package server

import (
	"fmt"
	"strconv"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/deployment"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/deploy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrorf = grpc.Errorf

var (
	// ErrNotImplemented is returned from RPC calls that have yet to be implemented
	ErrNotImplemented = errors.New("Not Implemented")
)

type server struct {
	dl      dal.DAL
	manager *deployment.Manager
}

// New returns an initialized instance of server
func New(dl dal.DAL, manager *deployment.Manager) deploy.DeployServer {
	return &server{
		dl:      dl,
		manager: manager,
	}
}

// CreateDeployable creates a single deployable object for a given deployable group
func (s *server) CreateDeployable(ctx context.Context, in *deploy.CreateDeployableRequest) (*deploy.CreateDeployableResponse, error) {
	if in.DeployableGroupID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "group id cannot be empty")
	}
	groupID, err := dal.ParseDeployableGroupID(in.DeployableGroupID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "group id %q is invalid", in.DeployableGroupID)
	}
	if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name cannot be empty")
	}

	if _, err := s.dl.DeployableGroup(groupID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found: Deployable Group: %q", groupID)
		}
		return nil, errors.Trace(err)
	}

	if _, err := s.dl.DeployableForNameAndGroup(in.Name, groupID); err == nil {
		return nil, grpcErrorf(codes.InvalidArgument, "name %s is not available for this group", in.Name)
	} else if errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	}

	id, err := s.dl.InsertDeployable(&dal.Deployable{
		Name:              in.Name,
		Description:       in.Description,
		DeployableGroupID: groupID,
		GitURL:            in.GitURL,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	dep, err := s.dl.Deployable(id)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.CreateDeployableResponse{
		Deployable: transformDeployableToResponse(dep),
	}, nil
}

// CreateDeployableConfig creates a versioned config set for a given environment/deployable
func (s *server) CreateDeployableConfig(ctx context.Context, in *deploy.CreateDeployableConfigRequest) (*deploy.CreateDeployableConfigResponse, error) {
	if in.DeployableID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id cannot be empty")
	}
	depID, err := dal.ParseDeployableID(in.DeployableID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id %q is invalid", in.DeployableID)
	}
	if in.EnvironmentID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "environment id cannot be empty")
	}
	envID, err := dal.ParseEnvironmentID(in.EnvironmentID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "environment id %q is invalid", in.EnvironmentID)
	}
	if _, err = s.dl.Deployable(depID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found: Deployable: %q", depID)
		}
		return nil, errors.Trace(err)
	}
	if _, err = s.dl.Environment(envID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found: Environment: %q", envID)
		}
		return nil, errors.Trace(err)
	}
	omitMap := make(map[string]struct{})
	for _, omit := range in.OmitFromSource {
		omitMap[omit] = struct{}{}
	}

	configMap := make(map[string]string)
	if in.SourceConfigID != "" {
		sourceConfigID, err := dal.ParseDeployableConfigID(in.SourceConfigID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "source deployable config id %q is invalid", in.SourceConfigID)
		}
		if _, err = s.dl.DeployableConfig(sourceConfigID); err != nil {
			if errors.Cause(err) == dal.ErrNotFound {
				return nil, grpcErrorf(codes.NotFound, "Not Found: Deployable Config: %q", sourceConfigID)
			}
			return nil, errors.Trace(err)
		}
		sourceConfigValues, err := s.dl.DeployableConfigValues(sourceConfigID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		for _, sv := range sourceConfigValues {
			if _, ok := omitMap[sv.Name]; !ok {
				configMap[sv.Name] = sv.Value
			}
		}
	}
	for n, v := range in.Values {
		configMap[n] = v
	}

	var configID dal.DeployableConfigID
	if err := s.dl.Transact(func(dl dal.DAL) error {
		if _, err := dl.DeprecateActiveDeployableConfig(depID, envID); err != nil {
			return errors.Trace(err)
		}
		configID, err = dl.InsertDeployableConfig(&dal.DeployableConfig{
			DeployableID:  depID,
			EnvironmentID: envID,
			// TODO: Add the ability to stage a config before making it active
			Status: dal.DeployableConfigStatusActive,
		})
		if err != nil {
			return errors.Trace(err)
		}

		var i int
		configValues := make([]*dal.DeployableConfigValue, len(configMap))
		for n, v := range configMap {
			configValues[i] = &dal.DeployableConfigValue{
				DeployableConfigID: configID,
				Name:               n,
				Value:              v,
			}
			i++
		}

		return dl.InsertDeployableConfigValues(configValues)
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.CreateDeployableConfigResponse{
		Config: &deploy.DeployableConfig{
			ID:            configID.String(),
			DeployableID:  depID.String(),
			EnvironmentID: envID.String(),
			Status:        string(dal.DeployableConfigStatusActive),
			Values:        configMap,
		},
	}, nil
}

// CreateDeployableGroup creates a logical group for encapsulating lets of deployables
func (s *server) CreateDeployableGroup(ctx context.Context, in *deploy.CreateDeployableGroupRequest) (*deploy.CreateDeployableGroupResponse, error) {
	if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name cannot be empty")
	}

	if _, err := s.dl.DeployableGroupForName(in.Name); err == nil {
		return nil, grpcErrorf(codes.InvalidArgument, "group name %s is not available", in.Name)
	} else if errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	}

	id, err := s.dl.InsertDeployableGroup(&dal.DeployableGroup{
		Name:        in.Name,
		Description: in.Description,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	group, err := s.dl.DeployableGroup(id)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.CreateDeployableGroupResponse{
		DeployableGroup: transformDeployableGroupToResponse(group),
	}, nil
}

// CreateDeployableVector creates a single deployable vector for a given deployable
func (s *server) CreateDeployableVector(ctx context.Context, in *deploy.CreateDeployableVectorRequest) (*deploy.CreateDeployableVectorResponse, error) {
	if in.DeployableID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id cannot be empty")
	}
	depID, err := dal.ParseDeployableID(in.DeployableID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id %q is invalid", in.DeployableID)
	}
	if in.TargetEnvironmentID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "target environment id cannot be empty")
	}
	targetEnvID, err := dal.ParseEnvironmentID(in.TargetEnvironmentID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "environment id %q is invalid", in.TargetEnvironmentID)
	}
	vectorSourceType, err := dal.ParseDeployableVectorSourceType(in.SourceType.String())
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "%q is not a valid deployable vector source type", in.SourceType)
	}
	var sourceEnvID dal.EnvironmentID
	if vectorSourceType == dal.DeployableVectorSourceTypeEnvironmentID {
		if in.GetSourceEnvironmentID() == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "source environment id cannot be empty for source type %s", vectorSourceType)
		}
		sourceEnvID, err = dal.ParseEnvironmentID(in.GetSourceEnvironmentID())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "environment id %q is invalid", in.GetSourceEnvironmentID())
		}
	}

	dep, err := s.dl.Deployable(depID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found: Deployable: %q", depID)
		}
		return nil, errors.Trace(err)
	}
	tEnv, err := s.dl.Environment(targetEnvID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found: Environment: %q", targetEnvID)
		}
		return nil, errors.Trace(err)
	}
	if dep.DeployableGroupID != tEnv.DeployableGroupID {
		return nil, grpcErrorf(codes.InvalidArgument, "this deployable does have access to environment %q", targetEnvID)
	}
	if vectorSourceType == dal.DeployableVectorSourceTypeEnvironmentID {
		sEnv, err := s.dl.Environment(sourceEnvID)
		if err != nil {
			if errors.Cause(err) == dal.ErrNotFound {
				return nil, grpcErrorf(codes.NotFound, "Not Found: Environment: %q", sourceEnvID)
			}
			return nil, errors.Trace(err)
		}
		if dep.DeployableGroupID != sEnv.DeployableGroupID {
			return nil, grpcErrorf(codes.InvalidArgument, "this deployable does have access to environment %q", sourceEnvID)
		}
	}
	if _, err := s.dl.DeployableVectorForDeployableSourceTarget(depID, vectorSourceType, sourceEnvID, targetEnvID); err == nil {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable vector for deployable %s source type %s source env %s target env %s already exists", depID, vectorSourceType, sourceEnvID, targetEnvID)
	} else if errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	}

	id, err := s.dl.InsertDeployableVector(&dal.DeployableVector{
		DeployableID:        depID,
		SourceType:          vectorSourceType,
		SourceEnvironmentID: sourceEnvID,
		TargetEnvironmentID: targetEnvID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	dv, err := s.dl.DeployableVector(id)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.CreateDeployableVectorResponse{
		Vector: transformDeployableVectorToResponse(dv),
	}, nil
}

// CreateEnvironment creates a stage for a given deployable group
func (s *server) CreateEnvironment(ctx context.Context, in *deploy.CreateEnvironmentRequest) (*deploy.CreateEnvironmentResponse, error) {
	if in.DeployableGroupID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "group id cannot be empty")
	}
	groupID, err := dal.ParseDeployableGroupID(in.DeployableGroupID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "group id %q is invalid", in.DeployableGroupID)
	}
	if in.Name == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "name cannot be empty")
	}

	if _, err = s.dl.DeployableGroup(groupID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found: Deployable Group: %q", groupID)
		}
		return nil, errors.Trace(err)
	}

	if _, err := s.dl.EnvironmentForNameAndGroup(in.Name, groupID); err == nil {
		return nil, grpcErrorf(codes.InvalidArgument, "name %s is not available for this group", in.Name)
	} else if errors.Cause(err) != dal.ErrNotFound {
		return nil, errors.Trace(err)
	}

	id, err := s.dl.InsertEnvironment(&dal.Environment{
		Name:              in.Name,
		Description:       in.Description,
		IsProd:            in.IsProd,
		DeployableGroupID: groupID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	env, err := s.dl.Environment(id)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.CreateEnvironmentResponse{
		Environment: transformEnvironmentToResponse(env),
	}, nil
}

// TODO: See if we can merge some logic for deployable configs
// CreateEnvironmentConfig creates a versioned config set for a given environment
func (s *server) CreateEnvironmentConfig(ctx context.Context, in *deploy.CreateEnvironmentConfigRequest) (*deploy.CreateEnvironmentConfigResponse, error) {
	if in.EnvironmentID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "environment id cannot be empty")
	}
	envID, err := dal.ParseEnvironmentID(in.EnvironmentID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "environment id %q is invalid", in.EnvironmentID)
	}
	_, err = s.dl.Environment(envID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found: Environment: %q", envID)
		}
		return nil, errors.Trace(err)
	}
	omitMap := make(map[string]struct{})
	for _, omit := range in.OmitFromSource {
		omitMap[omit] = struct{}{}
	}

	configMap := make(map[string]string)
	if in.SourceConfigID != "" {
		sourceConfigID, err := dal.ParseEnvironmentConfigID(in.SourceConfigID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "source environment config id %q is invalid", in.SourceConfigID)
		}
		if _, err = s.dl.EnvironmentConfig(sourceConfigID); err != nil {
			if errors.Cause(err) == dal.ErrNotFound {
				return nil, grpcErrorf(codes.NotFound, "Not Found: Environment Config: %q", sourceConfigID)
			}
			return nil, errors.Trace(err)
		}
		sourceConfigValues, err := s.dl.EnvironmentConfigValues(sourceConfigID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		for _, sv := range sourceConfigValues {
			if _, ok := omitMap[sv.Name]; !ok {
				configMap[sv.Name] = sv.Value
			}
		}
	}
	for n, v := range in.Values {
		configMap[n] = v
	}

	var configID dal.EnvironmentConfigID
	if err := s.dl.Transact(func(dl dal.DAL) error {
		if _, err := dl.DeprecateActiveEnvironmentConfig(envID); err != nil {
			return errors.Trace(err)
		}
		configID, err = dl.InsertEnvironmentConfig(&dal.EnvironmentConfig{
			EnvironmentID: envID,
			// TODO: Add the ability to stage a config before making it active
			Status: dal.EnvironmentConfigStatusActive,
		})
		if err != nil {
			return errors.Trace(err)
		}

		var i int
		configValues := make([]*dal.EnvironmentConfigValue, len(configMap))
		for n, v := range configMap {
			configValues[i] = &dal.EnvironmentConfigValue{
				EnvironmentConfigID: configID,
				Name:                n,
				Value:               v,
			}
			i++
		}

		return dl.InsertEnvironmentConfigValues(configValues)
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.CreateEnvironmentConfigResponse{
		Config: &deploy.EnvironmentConfig{
			ID:            configID.String(),
			EnvironmentID: envID.String(),
			Status:        string(dal.EnvironmentConfigStatusActive),
			Values:        configMap,
		},
	}, nil
}

// DeployableConfigs returns all the deployable configs for a given environment and deployable
func (s *server) DeployableConfigs(ctx context.Context, in *deploy.DeployableConfigsRequest) (*deploy.DeployableConfigsResponse, error) {
	if in.DeployableID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id cannot be empty")
	}
	depID, err := dal.ParseDeployableID(in.DeployableID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id %q is invalid", in.DeployableID)
	}
	if in.EnvironmentID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "group id cannot be empty")
	}
	envID, err := dal.ParseEnvironmentID(in.EnvironmentID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "environment id %q is invalid", in.EnvironmentID)
	}
	status, err := dal.ParseDeployableConfigStatus(in.Status)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "%q is not a valid config status", in.Status)
	}

	depConfigs, err := s.dl.DeployableConfigsForStatus(depID, envID, status)
	if err != nil {
		return nil, errors.Trace(err)
	}

	deployableConfigs := make([]*deploy.DeployableConfig, len(depConfigs))
	for i, dc := range depConfigs {
		configMap := make(map[string]string)
		configValues, err := s.dl.DeployableConfigValues(dc.ID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		for _, v := range configValues {
			configMap[v.Name] = v.Value
		}
		deployableConfigs[i] = transformDeployableConfigToResponse(dc, configMap)
	}

	return &deploy.DeployableConfigsResponse{
		Configs: deployableConfigs,
	}, nil
}

// DeployableGroups returns all the deployable groups in the system
func (s *server) DeployableGroups(ctx context.Context, in *deploy.DeployableGroupsRequest) (*deploy.DeployableGroupsResponse, error) {
	groups, err := s.dl.DeployableGroups()
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.DeployableGroupsResponse{
		DeployableGroups: transformDeployableGroupsToResponse(groups),
	}, nil
}

// Deployments returns all the deployments for a given deployable group and status
func (s *server) Deployments(ctx context.Context, in *deploy.DeploymentsRequest) (*deploy.DeploymentsResponse, error) {
	if in.DeployableID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id cannot be empty")
	}
	depID, err := dal.ParseDeployableID(in.DeployableID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id %q is invalid", in.DeployableID)
	}

	if _, err = s.dl.Deployable(depID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "Not Found: Deployable: %q", depID)
		}
		return nil, errors.Trace(err)
	}

	var deployments []*dal.Deployment
	switch in.Status {
	case deploy.DeploymentsRequest_ANY:
		deployments, err = s.dl.Deployments(depID)
	default:
		ds, err := dal.ParseDeploymentStatus(in.Status.String())
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "deployment status %q is invalid", in.Status.String())
		}
		deployments, err = s.dl.DeploymentsForStatus(depID, ds)
	}

	rDeployments, err := transformDeploymentsToResponse(deployments)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.DeploymentsResponse{
		Deployments: rDeployments,
	}, nil
}

// Deployables returns all the deployables for a given deployable group
func (s *server) Deployables(ctx context.Context, in *deploy.DeployablesRequest) (*deploy.DeployablesResponse, error) {
	switch by := in.By.(type) {
	case *deploy.DeployablesRequest_DeployableID:
		if by.DeployableID == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "deployable id cannot be empty")
		}
		depID, err := dal.ParseDeployableID(by.DeployableID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "deployable id %q is invalid", by.DeployableID)
		}
		dep, err := s.dl.Deployable(depID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "deployable id %q not found", by.DeployableID)
		}
		if err != nil {
			return nil, errors.Trace(err)
		}
		return &deploy.DeployablesResponse{
			Deployables: transformDeployablesToResponse([]*dal.Deployable{dep}),
		}, nil
	case *deploy.DeployablesRequest_DeployableGroupID:
		if by.DeployableGroupID == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "group id cannot be empty")
		}
		groupID, err := dal.ParseDeployableGroupID(by.DeployableGroupID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "group id %q is invalid", by.DeployableGroupID)
		}
		deps, err := s.dl.DeployablesForGroup(groupID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return &deploy.DeployablesResponse{
			Deployables: transformDeployablesToResponse(deps),
		}, nil
	}
	return nil, errors.Errorf("Unsupported by type %T", in.By)
}

// DeployableVectors returns all the deployable vectors for a given deployable
func (s *server) DeployableVectors(ctx context.Context, in *deploy.DeployableVectorsRequest) (*deploy.DeployableVectorsResponse, error) {
	if in.DeployableID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id cannot be empty")
	}
	depID, err := dal.ParseDeployableID(in.DeployableID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id %q is invalid", in.DeployableID)
	}

	vectors, err := s.dl.DeployableVectorsForDeployable(depID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &deploy.DeployableVectorsResponse{
		Vectors: transformDeployableVectorsToResponse(vectors),
	}, nil
}

// EnvironmentConfigs returns all the environment configs for a given environment
func (s *server) EnvironmentConfigs(ctx context.Context, in *deploy.EnvironmentConfigsRequest) (*deploy.EnvironmentConfigsResponse, error) {
	if in.EnvironmentID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "group id cannot be empty")
	}
	envID, err := dal.ParseEnvironmentID(in.EnvironmentID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "environment id %q is invalid", in.EnvironmentID)
	}
	status, err := dal.ParseEnvironmentConfigStatus(in.Status)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "%q is not a valid config status", in.Status)
	}

	envConfigs, err := s.dl.EnvironmentConfigsForStatus(envID, status)
	if err != nil {
		return nil, errors.Trace(err)
	}

	environmentConfigs := make([]*deploy.EnvironmentConfig, len(envConfigs))
	for i, ec := range envConfigs {
		configMap := make(map[string]string)
		configValues, err := s.dl.EnvironmentConfigValues(ec.ID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		for _, v := range configValues {
			configMap[v.Name] = v.Value
		}
		environmentConfigs[i] = transformEnvironmentConfigToResponse(ec, configMap)
	}

	return &deploy.EnvironmentConfigsResponse{
		Configs: environmentConfigs,
	}, nil
}

// Environments returns all the environments for a given deployable group
func (s *server) Environments(ctx context.Context, in *deploy.EnvironmentsRequest) (*deploy.EnvironmentsResponse, error) {
	switch by := in.By.(type) {
	case *deploy.EnvironmentsRequest_EnvironmentID:
		if by.EnvironmentID == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "environment id cannot be empty")
		}
		envID, err := dal.ParseEnvironmentID(by.EnvironmentID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "environment id %q is invalid", by.EnvironmentID)
		}
		env, err := s.dl.Environment(envID)
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpcErrorf(codes.NotFound, "environment id %q not found", by.EnvironmentID)
		}
		if err != nil {
			return nil, errors.Trace(err)
		}
		return &deploy.EnvironmentsResponse{
			Environments: transformEnvironmentsToResponse([]*dal.Environment{env}),
		}, nil
	case *deploy.EnvironmentsRequest_DeployableGroupID:
		if by.DeployableGroupID == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "group id cannot be empty")
		}
		groupID, err := dal.ParseDeployableGroupID(by.DeployableGroupID)
		if err != nil {
			return nil, grpcErrorf(codes.InvalidArgument, "group id %q is invalid", by.DeployableGroupID)
		}
		envs, err := s.dl.EnvironmentsForGroup(groupID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return &deploy.EnvironmentsResponse{
			Environments: transformEnvironmentsToResponse(envs),
		}, nil
	}
	return nil, errors.Errorf("Unsupported by type %T", in.By)
}

// Promote reports that a deployable or deployable group should be promoted to all available outbound vectors
func (s *server) Promote(ctx context.Context, in *deploy.PromotionRequest) (*deploy.PromotionResponse, error) {
	if in.DeploymentID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "deployment id cannot be empty")
	}

	deploymentIDs, err := s.manager.ProcessPromotionEvent(&deploy.PromotionEvent{
		DeploymentID: in.DeploymentID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	deployments := make([]*dal.Deployment, len(deploymentIDs))
	for i, dID := range deploymentIDs {
		deployments[i], err = s.dl.Deployment(dID)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	rDeployments, err := transformDeploymentsToResponse(deployments)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &deploy.PromotionResponse{
		Deployments: rDeployments,
	}, nil
}

// PromoteGroup reports that all deployments of a deployable group with a given build number should be promoted out of the target environment
func (s *server) PromoteGroup(ctx context.Context, in *deploy.PromoteGroupRequest) (*deploy.PromoteGroupResponse, error) {
	if in.DeployableGroupID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable group id cannot be empty")
	}
	depID, err := dal.ParseDeployableGroupID(in.DeployableGroupID)
	if err != nil {
		return nil, fmt.Errorf("deployable group id %q is invalid", in.DeployableGroupID)
	}
	if _, err = s.dl.DeployableGroup(depID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, fmt.Errorf("Not Found: Deployable Group: %q", depID)
		}
		return nil, err
	}

	if in.BuildNumber == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "build number cannot be empty")
	}
	if _, err := strconv.ParseInt(in.BuildNumber, 10, 64); err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "cannot parse build number %s: %s", in.BuildNumber, err)
	}

	if in.EnvironmentID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "environment id cannot be empty")
	}
	envID, err := dal.ParseEnvironmentID(in.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("environment id %q is invalid", in.EnvironmentID)
	}
	if _, err = s.dl.Environment(envID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, fmt.Errorf("Not Found: Environment: %q", envID)
		}
		return nil, err
	}

	depls, err := s.dl.DeploymentsForDeploymentGroup(depID, envID, in.BuildNumber)
	if err != nil {
		return nil, err
	}
	if len(depls) == 0 {
		return nil, grpcErrorf(codes.FailedPrecondition, "No existing deployments matching the criteria were found")
	}

	var deploymentIDs []dal.DeploymentID
	for _, depl := range depls {
		dIDs, err := s.manager.ProcessPromotionEvent(&deploy.PromotionEvent{
			DeploymentID: depl.ID.String(),
		})
		if err != nil {
			return nil, err
		}
		deploymentIDs = append(deploymentIDs, dIDs...)
	}
	if err != nil {
		return nil, errors.Trace(err)
	}

	deployments := make([]*dal.Deployment, len(deploymentIDs))
	for i, dID := range deploymentIDs {
		deployments[i], err = s.dl.Deployment(dID)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	rDeployments, err := transformDeploymentsToResponse(deployments)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &deploy.PromoteGroupResponse{
		Deployments: rDeployments,
	}, nil
}

// ReportBuildComplete reports the completion of a build for deployment
func (s *server) ReportBuildComplete(ctx context.Context, in *deploy.ReportBuildCompleteRequest) (*deploy.ReportBuildCompleteResponse, error) {
	ev := &deploy.BuildCompleteEvent{}
	if in.DeployableID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "deployable id cannot be empty")
	}
	ev.DeployableID = in.DeployableID
	if in.BuildNumber == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "build number cannot be empty")
	}
	ev.GitHash = in.GitHash
	if in.GitHash == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "git hash cannot be empty")
	}
	ev.BuildNumber = in.BuildNumber

	switch in.ArtifactType {
	case deploy.ReportBuildCompleteRequest_DOCKER_IMAGE:
		if in.GetDockerImage() == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "image cannot be empty for artifacts of type %s", deploy.ReportBuildCompleteRequest_DOCKER_IMAGE.String())
		}
		ev.Image = in.GetDockerImage()
	default:
		return nil, grpcErrorf(codes.InvalidArgument, "unknown artifact type", in.ArtifactType.String())
	}

	deploymentIDs, err := s.manager.ProcessBuildCompleteEvent(ev)
	if err != nil {
		return nil, errors.Trace(err)
	}

	deployments := make([]*dal.Deployment, len(deploymentIDs))
	for i, dID := range deploymentIDs {
		deployments[i], err = s.dl.Deployment(dID)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	rDeployments, err := transformDeploymentsToResponse(deployments)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &deploy.ReportBuildCompleteResponse{
		Deployments: rDeployments,
	}, nil
}
