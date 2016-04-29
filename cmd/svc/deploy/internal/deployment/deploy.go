package deployment

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/deploy/internal/dal"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/slack"
)

const (
	deploymentWebhookURL = "https://hooks.slack.com/services/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"
	deployUserName       = "deploy"
	deployChannel        = "#x-backend-deployments"
	deployGoodEmoji      = ":construction_worker:"
	deployBadEmoji       = ":boom:"
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

func postStartMessage(depl *dal.Deployment, dep *dal.Deployable, env *dal.Environment) {
	if err := slack.Post(deploymentWebhookURL, &slack.Message{
		Text:      fmt.Sprintf("`STARTING` deployment for `%s:%s` to environment `%s`", dep.Name, depl.BuildNumber, env.Name),
		Username:  deployUserName,
		Channel:   deployChannel,
		IconEmoji: deployGoodEmoji,
	}); err != nil {
		golog.Errorf("Failed to post start message to slack: %s", err)
	}
}

func postCompleteMessage(depl *dal.Deployment, dep *dal.Deployable, env *dal.Environment) {
	if err := slack.Post(deploymentWebhookURL, &slack.Message{
		Text:      fmt.Sprintf("`COMPLETED` deployment for `%s:%s` to environment `%s`", dep.Name, depl.BuildNumber, env.Name),
		Username:  deployUserName,
		Channel:   deployChannel,
		IconEmoji: deployGoodEmoji,
	}); err != nil {
		golog.Errorf("Failed to post completed message to slack: %s", err)
	}
}

func postFailedMessage(depl *dal.Deployment, dep *dal.Deployable, env *dal.Environment, err error) {
	if err := slack.Post(deploymentWebhookURL, &slack.Message{
		Text:      fmt.Sprintf("`FAILED` deployment for `%s:%s` to environment `%s`- `%s`", dep.Name, depl.BuildNumber, env.Name, err),
		Username:  deployUserName,
		Channel:   deployChannel,
		IconEmoji: deployBadEmoji,
	}); err != nil {
		golog.Errorf("Failed to post failed message to slack: %s", err)
	}
}

func (m *Manager) processDeployment(depl *dal.Deployment) error {
	if depl == nil {
		return nil
	}

	dep, err := m.dl.Deployable(depl.DeployableID)
	if err != nil {
		return err
	}
	env, err := m.dl.Environment(depl.EnvironmentID)
	if err != nil {
		return err
	}

	// dispatch the deployment based on the type
	switch depl.Type {
	case dal.DeploymentTypeEcs:
		conc.Go(func() {
			postStartMessage(depl, dep, env)
			if err := m.processECSDeployment(depl); err != nil {
				postFailedMessage(depl, dep, env, err)
				m.failDeployment(depl.ID, err)
			} else {
				postCompleteMessage(depl, dep, env)
				m.completeDeployment(depl.ID)
			}
		})
	default:
		return fmt.Errorf("Unknown deployment type %s", depl.Type)
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
	if err := m.dl.SetDeploymentStatus(id, dal.DeploymentStatusComplete); err != nil {
		golog.Errorf("Encountered error while marking deployment %s as COMPLETE: %s", id, err)
	}
	return
}
