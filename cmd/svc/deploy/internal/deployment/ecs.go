package deployment

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/dal"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/deploy"
)

func (m *Manager) processECSDeployment(d *dal.Deployment) error {
	rTDefInput, err := m.taskDefinitionInputForDeployment(d)
	if err != nil {
		return err
	}
	env, err := m.dl.Environment(d.EnvironmentID)
	if err != nil {
		return err
	}
	// Assume the correct role. For now hack this.
	// TODO: Figure out how to track roles vs envs
	// If it's not prod, assume the dev role. This is the jank.
	var ecsCli ecsiface.ECSAPI
	if !env.IsProd {
		ecsCli, err = awsutil.AssumedECSCli(m.stsCli, "arn:aws:iam::758505115169:role/dev-deploy-ecs", d.ID.String())
		if err != nil {
			return err
		}
	} else {
		ecsCli = ecs.New(m.awsSession)
	}

	res, err := ecsCli.RegisterTaskDefinition(rTDefInput)
	if err != nil {
		return err
	}
	golog.Infof("Registered Task Definition %s:%d", *res.TaskDefinition.Family, *res.TaskDefinition.Revision)

	// TODO: Starting to take some shortcuts from here out for the sake of time, all marked with TODO
	dep, err := m.dl.Deployable(d.DeployableID)
	if err != nil {
		return err
	}
	uRes, err := ecsCli.UpdateService(&ecs.UpdateServiceInput{
		//TODO: Get the service name from the deployable config somehow
		Cluster:        ptr.String(fmt.Sprintf("%s-svc", env.Name)),
		Service:        ptr.String(dep.Name),
		TaskDefinition: ptr.String(fmt.Sprintf("%s:%d", *res.TaskDefinition.Family, *res.TaskDefinition.Revision)),
	})
	if err != nil {
		return err
	}
	golog.Infof("Updated Service %s in Cluster %s with Task Definition %s", *uRes.Service.ServiceName, *uRes.Service.ClusterArn, *uRes.Service.TaskDefinition)
	return nil
}

func ecsConfigName(sub string) string {
	return `ECS_CONFIG_` + strings.ToUpper(sub)
}

func splitStringPtrList(s string) []*string {
	vals := strings.Split(s, ",")
	strs := make([]*string, len(vals))
	for i, v := range vals {
		strs[i] = ptr.String(strings.TrimSpace(v))
	}
	return strs
}

// TODO: How to handle multiple containers in a task?
func (m *Manager) taskDefinitionInputForDeployment(d *dal.Deployment) (*ecs.RegisterTaskDefinitionInput, error) {
	// Validate our config
	dep, err := m.dl.Deployable(d.DeployableID)
	if err != nil {
		return nil, err
	}
	env, err := m.dl.Environment(d.EnvironmentID)
	if err != nil {
		return nil, err
	}
	dConfigs, err := m.dl.DeployableConfigValues(d.DeployableConfigID)
	if err != nil {
		return nil, err
	}
	eConfigs, err := m.dl.EnvironmentConfigsForStatus(d.EnvironmentID, dal.EnvironmentConfigStatusActive)
	if err != nil {
		return nil, err
	}
	var eConfigValues []*dal.EnvironmentConfigValue
	if len(eConfigs) != 0 {
		if len(eConfigs) > 1 {
			golog.Errorf("More than one active environment config for %s, using first", d.EnvironmentID)
		}
		eConfigValues, err = m.dl.EnvironmentConfigValues(eConfigs[0].ID)
		if err != nil {
			return nil, err
		}
	}

	// Default to environment config values and overwrite with deployable config values
	configs := make(map[string]string, len(dConfigs)+len(eConfigValues))
	envConfigReplacements := strings.NewReplacer("{deployname}", dep.Name, "{DEPLOYNAME}", strings.ToUpper(dep.Name))
	for _, c := range eConfigValues {
		configs[envConfigReplacements.Replace(c.Name)] = envConfigReplacements.Replace(c.Value)
	}
	for _, c := range dConfigs {
		configs[c.Name] = c.Value
	}

	awslogsGroup := env.Name + "-service"

	var portMappings []*ecs.PortMapping
	var mountPoints []*ecs.MountPoint
	var dnsServers []*string
	var dnsSearchDomains []*string
	var entryPoint []*string
	var command []*string
	var taskRoleARN *string
	var volumes []*ecs.Volume
	cMap := make(map[string]string, len(configs))
	for name, value := range configs {
		switch {
		case strings.HasPrefix(name, ecsConfigName("PORT_MAPPING")):
			pm, err := parsePortMapping(value)
			if err != nil {
				return nil, err
			}
			portMappings = append(portMappings, pm)
		case strings.HasPrefix(name, ecsConfigName("MOUNT_POINT")):
			mp, err := parseMountPoint(value)
			if err != nil {
				return nil, err
			}
			mountPoints = append(mountPoints, mp)
		case strings.HasPrefix(name, ecsConfigName("VOLUME")):
			v, err := parseVolume(value)
			if err != nil {
				return nil, err
			}
			volumes = append(volumes, v)
		case name == ecsConfigName("DNS_SERVERS"):
			dnsServers = splitStringPtrList(value)
		case name == ecsConfigName("DNS_SEARCH_DOMAINS"):
			dnsSearchDomains = splitStringPtrList(value)
		case name == ecsConfigName("ENTRY_POINT"):
			entryPoint = splitStringPtrList(value)
		case name == ecsConfigName("COMMAND"):
			command = splitStringPtrList(value)
		case name == ecsConfigName("TASK_ROLE_ARN"):
			taskRoleARN = ptr.String(value)
		case name == ecsConfigName("AWSLOGS_GROUP"):
			awslogsGroup = value
		default:
			cMap[name] = value
		}
	}

	cpu, err := fecthAndDeleteRequiredECSConfigInt64(cMap, ecsConfigName("cpu"))
	if err != nil {
		return nil, err
	}
	memory, err := fecthAndDeleteRequiredECSConfigInt64(cMap, ecsConfigName("memory"))
	if err != nil {
		return nil, err
	}
	var envVariables []*ecs.KeyValuePair
	for n, v := range cMap {
		envVariables = append(envVariables, &ecs.KeyValuePair{
			Name:  ptr.String(strings.ToUpper(n)),
			Value: ptr.String(v),
		})
	}
	ecsDeployment := &deploy.ECSDeployment{}
	if err := ecsDeployment.Unmarshal(d.Data); err != nil {
		return nil, err
	}
	return &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Image:       ptr.String(ecsDeployment.Image),
				Cpu:         &cpu,
				Memory:      &memory,
				EntryPoint:  entryPoint,
				Command:     command,
				Environment: envVariables,
				// TODO: Figure out how to manage this with multiple containers in task
				Essential:        ptr.Bool(true),
				Name:             ptr.String(dep.Name),
				PortMappings:     portMappings,
				MountPoints:      mountPoints,
				DnsServers:       dnsServers,
				DnsSearchDomains: dnsSearchDomains,
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: ptr.String("awslogs"),
					Options: map[string]*string{
						"awslogs-group":         &awslogsGroup,
						"awslogs-region":        ptr.String("us-east-1"), // TODO: dynamically set this once we do multi-region
						"awslogs-stream-prefix": &dep.Name,
					},
				},
			},
		},
		Family:      ptr.String(fmt.Sprintf("%s-%s", env.Name, dep.Name)),
		TaskRoleArn: taskRoleARN,
		Volumes:     volumes,
	}, nil
}

// parseVolume parses a task volume in the format "name:source path" with source path being optional
func parseVolume(v string) (*ecs.Volume, error) {
	parts := strings.Split(strings.TrimSpace(v), ":")
	if (len(parts) != 2 && len(parts) != 1) || parts[0] == "" {
		return nil, fmt.Errorf("%s is not a valid volume of format name[:path]", v)
	}
	// No source path means docker creates a new path on the host
	if len(parts) == 1 || parts[1] == "" {
		return &ecs.Volume{
			Name: &parts[0],
		}, nil
	}
	return &ecs.Volume{
		Name: &parts[0],
		Host: &ecs.HostVolumeProperties{
			SourcePath: &parts[1],
		},
	}, nil
}

// parseMountPoint parses a container mount point in the format "source volume:container path:read only"
func parseMountPoint(mp string) (*ecs.MountPoint, error) {
	parts := strings.Split(strings.TrimSpace(mp), ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[1][0] != '/' {
		return nil, fmt.Errorf("%s is not a valid mount point of format path:volume:readonly", mp)
	}
	readonly, err := strconv.ParseBool(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid mount point of format path:volume:readonly, readonly must be true or false", mp)
	}
	return &ecs.MountPoint{
		SourceVolume:  &parts[0],
		ContainerPath: &parts[1],
		ReadOnly:      &readonly,
	}, nil
}

// expected format: cport:hport:proto
func parsePortMapping(pm string) (*ecs.PortMapping, error) {
	ms := strings.Split(pm, ":")
	if len(ms) != 3 {
		return nil, fmt.Errorf("%s is not a valid ECS_PORT_MAPPING of format cport:hport:proto", pm)
	}
	cport, err := strconv.ParseInt(ms[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid ECS_PORT_MAPPING of format cport:hport:proto: %s", pm, err)
	}
	hport, err := strconv.ParseInt(ms[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid ECS_PORT_MAPPING of format cport:hport:proto: %s", pm, err)
	}
	proto := ms[2]
	return &ecs.PortMapping{
		ContainerPort: &cport,
		HostPort:      &hport,
		Protocol:      &proto,
	}, nil
}

func fecthAndDeleteRequiredECSConfigInt64(cMap map[string]string, name string) (int64, error) {
	v, ok := cMap[name]
	if !ok {
		return 0, fmt.Errorf("Deployable Config %s is required for ECS deployments. Have: %v", name, cMap)
	}
	delete(cMap, name)
	return strconv.ParseInt(v, 10, 64)
}
