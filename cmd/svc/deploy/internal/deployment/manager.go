package deployment

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/deploy"
)

// Manager represents the driver for deployments and event processing
type Manager struct {
	dl      dal.DAL
	ecsCli  ecsiface.ECSAPI
	stsCli  stsiface.STSAPI
	nWorker worker.Worker
	dWorker worker.Worker
}

// NewManager returns an initialized instance of Manager
func NewManager(dl dal.DAL, awsSession *session.Session, eventsQueueURL string) *Manager {
	m := &Manager{
		dl:     dl,
		ecsCli: ecs.New(awsSession),
		stsCli: sts.New(awsSession),
	}
	m.nWorker = newNotificationWorker(m, sqs.New(awsSession), eventsQueueURL)
	m.dWorker = worker.NewRepeat(time.Second*30, m.deploymentDiscovery)
	return m
}

// Start starts the service workers
func (m *Manager) Start() {
	m.StartDiscovery()
	m.StartEventReciever()
}

// Stop stops the service workers
func (m *Manager) Stop() {
	m.StopDiscovery()
	m.StopEventReciever()
}

// StartDiscovery starts the discovery of new pending deployments
func (m *Manager) StartDiscovery() {
	m.dWorker.Start()
}

// StopDiscovery stops the discovery of new pending deployments
func (m *Manager) StopDiscovery() {
	m.dWorker.Stop(time.Second * 5)
}

// StartEventReciever starts the event reciever for deployment events
func (m *Manager) StartEventReciever() {
	m.nWorker.Start()
}

// StopEventReciever stops the event reciever for deployment events
func (m *Manager) StopEventReciever() {
	m.nWorker.Stop(time.Second * 5)
}

// ProcessBuildCompleteEvent processes an event representing a completed event
func (m *Manager) ProcessBuildCompleteEvent(ev *deploy.BuildCompleteEvent) ([]dal.DeploymentID, error) {
	deploym, err := m.deploymentForBuildComplete(ev)
	if err != nil {
		return nil, err
	}

	vectors, err := m.dl.DeployableVectorsForDeployableAndSource(deploym.DeployableID, dal.DeployableVectorSourceTypeBuild)
	if err != nil {
		return nil, err
	} else if len(vectors) == 0 {
		golog.Warningf("Recieved build complete event for deployable %s but no deployable vectors exist with source BUILD")
		return nil, nil
	}

	ids := make([]dal.DeploymentID, len(vectors))
	if err := m.dl.Transact(func(dl dal.DAL) error {
		for i, v := range vectors {
			activeConfig, err := m.activeDepoyableConfig(deploym.DeployableID, v.TargetEnvironmentID)
			if err != nil {
				return err
			}
			id, err := dl.InsertDeployment(&dal.Deployment{
				Type:               deploym.Type,
				Data:               deploym.Data,
				Status:             deploym.Status,
				BuildNumber:        deploym.BuildNumber,
				DeployableID:       deploym.DeployableID,
				EnvironmentID:      v.TargetEnvironmentID,
				DeployableConfigID: activeConfig.ID,
				DeployableVectorID: v.ID,
				GitHash:            ev.GitHash,
			})
			if err != nil {
				return err
			}
			ids[i] = id
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return ids, nil
}

func (m *Manager) activeDepoyableConfig(depID dal.DeployableID, environmentID dal.EnvironmentID) (*dal.DeployableConfig, error) {
	configs, err := m.dl.DeployableConfigsForStatus(depID, environmentID, dal.DeployableConfigStatusActive)
	if err != nil {
		return nil, err
	}
	if len(configs) != 1 {
		return nil, fmt.Errorf("Expected 1 ACTIVE deployable config for deployable %s and environment %s but got %d", depID, environmentID, len(configs))
	}
	return configs[0], nil
}

func (m *Manager) deploymentForBuildComplete(ev *deploy.BuildCompleteEvent) (*dal.Deployment, error) {
	depID, err := dal.ParseDeployableID(ev.DeployableID)
	if err != nil {
		return nil, fmt.Errorf("deployable id %q is invalid", ev.DeployableID)
	}
	if _, err = m.dl.Deployable(depID); err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, fmt.Errorf("Not Found: Deployable: %q", depID)
		}
		return nil, err
	}
	if ev.BuildNumber == "" {
		return nil, fmt.Errorf("build number %q is required", ev.DeployableID)
	}
	deploym := &dal.Deployment{
		DeployableID: depID,
		BuildNumber:  ev.BuildNumber,
		Status:       dal.DeploymentStatusPending,
	}
	depData := &deploy.ECSDeployment{
		Image: ev.Image,
	}
	data, err := depData.Marshal()
	if err != nil {
		return nil, err
	}
	deploym.Data = data
	deploym.Type = dal.DeploymentTypeEcs
	return deploym, nil
}
