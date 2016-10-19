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

	var portMappings []*ecs.PortMapping
	var dnsServers []*string
	var dnsSearchDomains []*string
	var entryPoint []*string
	var command []*string
	cMap := make(map[string]string, len(dConfigs))
	for _, c := range dConfigs {
		switch {
		case strings.HasPrefix(c.Name, ecsConfigName("PORT_MAPPING")):
			pm, err := parsePortMapping(c.Value)
			if err != nil {
				return nil, err
			}
			portMappings = append(portMappings, pm)
		case strings.HasPrefix(c.Name, ecsConfigName("DNS_SERVERS")):
			dnsServers = splitStringPtrList(c.Value)
		case strings.HasPrefix(c.Name, ecsConfigName("DNS_SEARCH_DOMAINS")):
			dnsSearchDomains = splitStringPtrList(c.Value)
		case strings.HasPrefix(c.Name, ecsConfigName("ENTRY_POINT")):
			entryPoint = splitStringPtrList(c.Value)
		case strings.HasPrefix(c.Name, ecsConfigName("COMMAND")):
			command = splitStringPtrList(c.Value)
		default:
			cMap[c.Name] = c.Value
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
				DnsServers:       dnsServers,
				DnsSearchDomains: dnsSearchDomains,
				LogConfiguration: &ecs.LogConfiguration{
					LogDriver: ptr.String("awslogs"),
					Options: map[string]*string{
						"awslogs-group":  ptr.String(fmt.Sprintf("%s-%s", env.Name, dep.Name)),
						"awslogs-region": ptr.String("us-east-1"), // TODO: dynamically set this once we do multi-region
					},
				},
			},
		},
		Family: ptr.String(fmt.Sprintf("%s-%s", env.Name, dep.Name)),
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
