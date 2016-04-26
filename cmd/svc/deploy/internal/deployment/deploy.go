package deployment

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/dal"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

func (m *Manager) deploymentDiscovery() {
	golog.Debugf("Looking for pending deployments to begin...")
	var err error
	var dep *dal.Deployment
	if err := m.dl.Transact(func(dl dal.DAL) error {
		dep, err = dl.NextPendingDeployment()
		if errors.Cause(err) == dal.ErrNotFound {
			golog.Debugf("No deployments currently pending...")
			return nil
		} else if err != nil {
			return err
		}
		golog.Debugf("Discovered pending deployment #%d %s of Type: %s", dep.DeploymentNumber, dep.ID, dep.Type)

		// Take the lock for this deployment
		return dl.SetDeploymentStatus(dep.ID, dal.DeploymentStatusInProgress)
	}); err != nil {
		golog.Errorf("Encountered error while discovering pending deployments: %s", err)
		return
	}

	if err := m.processDeployment(dep); err != nil {
		m.failDeployment(dep.ID, err)
	}
}

func (m *Manager) processDeployment(dep *dal.Deployment) error {
	if dep == nil {
		return nil
	}
	// dispatch the deployment based on the type
	switch dep.Type {
	case dal.DeploymentTypeEcs:
		conc.Go(func() {
			if err := m.processECSDeployment(dep); err != nil {
				m.failDeployment(dep.ID, err)
			} else {
				m.completeDeployment(dep.ID)
			}
		})
	default:
		return fmt.Errorf("Unknown deployment type %s", dep.Type)
	}
	return nil
}

func (m *Manager) failDeployment(id dal.DeploymentID, err error) {
	golog.Errorf("Encountered error while processing deployment %s: %s", id, err)
	golog.Errorf("Failing deployment %s", id)
	if err := m.dl.SetDeploymentStatus(id, dal.DeploymentStatusFailed); err != nil {
		golog.Errorf("Encountered error while marking deployment %s as FAILED: %s", id, err)
	}
	return
}

func (m *Manager) completeDeployment(id dal.DeploymentID) {
	golog.Infof("Completing deployment %s", id)
	if err := m.dl.SetDeploymentStatus(id, dal.DeploymentStatusFailed); err != nil {
		golog.Errorf("Encountered error while marking deployment %s as COMPLETE: %s", id, err)
	}
	return
}
