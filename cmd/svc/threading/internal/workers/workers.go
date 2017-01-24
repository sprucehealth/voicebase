package workers

import (
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/worker"
)

const workerErrMetricName = "ThreadingWorkerError"

// WorkerThreadClient represents a client that is consumed by the service workers
type WorkerThreadClient interface {
	setupThreadClient
	scheduledMessageThreadClient
	batchTasksThreadClient
}

// Workers collection of all workers used by the Threading system
type Workers struct {
	worker.Collection
	dal          dal.DAL
	clk          clock.Clock
	threadingCli WorkerThreadClient
}

// New initializes a collection of all workers used by the Threading system
func New(
	dl dal.DAL,
	sqs sqsiface.SQSAPI,
	threadingCli WorkerThreadClient,
	eventQueueURL string) *Workers {
	w := &Workers{
		dal:          dl,
		clk:          clock.New(),
		threadingCli: threadingCli,
	}
	w.AddWorker(newSetupThreadWorker(sqs, threadingCli, eventQueueURL))
	w.AddWorker(worker.NewRepeat(time.Minute, w.processPendingScheduledMessage))
	w.AddWorker(worker.NewRepeat(time.Second*5, w.processPendingBatchTasks))
	return w
}
