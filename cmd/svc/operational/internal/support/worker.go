package support

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"context"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/operational"
	"github.com/sprucehealth/backend/svc/threading"
)

var (
	doctorTitles = map[string]struct{}{
		"DDS":    {},
		"DMD":    {},
		"DO":     {},
		"DPM":    {},
		"DVM":    {},
		"MBBS":   {},
		"MD":     {},
		"OD":     {},
		"PharmD": {},
		"PhD":    {},
		"PsyD":   {},
	}
	californiaLocation     *time.Location
	supportMessageTemplate *template.Template
	buffer                 bytes.Buffer
)

const (
	// spruce support starts at 7:30am PST
	spruceSupportStartHour   = 7
	spruceSupportStartMinute = 30

	// spruce support ends at 10:30pm PST
	spruceSupportEndHour   = 22
	spruceSupportEndMinute = 30

	postMessageThreshold = 4 * time.Minute

	// post delayed message after 9:00 am PST
	spruceSupportDelayedMessageHour = 9

	supportMessage = `Hi {{.ProviderName}} - I’m reaching out because my job is to help you get the most out of Spruce. What type of practice are you in?`
)

type Worker struct {
	sqs       sqsiface.SQSAPI
	threading threading.ThreadsClient
	directory directory.DirectoryClient
	worker    *awsutil.SQSWorker
	clock     clock.Clock
}

type snsMessage struct {
	Message []byte
}

type messageContext struct {
	ProviderName string
}

func init() {
	var err error
	californiaLocation, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}

	supportMessageTemplate, err = template.New("").Parse(supportMessage)
	if err != nil {
		panic(err)
	}
}

func NewWorker(
	sqs sqsiface.SQSAPI,
	threading threading.ThreadsClient,
	directory directory.DirectoryClient,
	queueURL string) *Worker {
	w := &Worker{
		sqs:       sqs,
		threading: threading,
		directory: directory,
		clock:     clock.New(),
	}

	w.worker = awsutil.NewSQSWorker(sqs, queueURL, w.processSNSEvent)
	return w
}

func (w *Worker) Start() {
	w.worker.Start()
}

func (w *Worker) Stop(wait time.Duration) {
	w.worker.Stop(wait)
}

func (w *Worker) processSNSEvent(ctx context.Context, msg string) error {
	var snsMsg snsMessage
	if err := json.Unmarshal([]byte(msg), &snsMsg); err != nil {
		golog.Errorf("Failed to unmarshal sns message: %s", err.Error())
		return nil
	}

	var event operational.NewOrgCreatedEvent
	if err := event.Unmarshal(snsMsg.Message); err != nil {
		golog.Errorf("Failed to unmarshal event: %s", err)
	}

	return w.processEvent(ctx, &event)
}

func (w *Worker) processEvent(ctx context.Context, event *operational.NewOrgCreatedEvent) error {
	orgCreationTime := time.Unix(event.OrgCreated, 0)
	currentTimePST := w.clock.Now().In(californiaLocation)

	if withinSupportHours(currentTimePST) {
		if w.clock.Now().Sub(orgCreationTime) >= postMessageThreshold {
			return w.postMessage(ctx, event)
		}
	}

	return awsutil.ErrRetryAfter(15 * time.Minute)
}

func (w *Worker) postMessage(ctx context.Context, event *operational.NewOrgCreatedEvent) error {
	// don't post message if thread's message count > 1
	res, err := w.threading.Thread(ctx, &threading.ThreadRequest{
		ThreadID: event.SpruceSupportThreadID,
	})
	if err != nil {
		return errors.Trace(err)
	} else if res.Thread == nil {
		return errors.Trace(fmt.Errorf("Expected 1 thread to be returned for %s but got none", event.SpruceSupportThreadID))
	} else if res.Thread.MessageCount > 1 {
		// nothing to do as a message has already been posted on the thread.
		return nil
	}

	// lookup entity via account id
	entityLookupRes, err := w.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: event.InitialProviderEntityID,
		},
	})
	if err != nil {
		return errors.Trace(err)
	} else if len(entityLookupRes.Entities) != 1 {
		return errors.Trace(fmt.Errorf("Expected 1 entity for entityID %s but got %d", event.InitialProviderEntityID, len(entityLookupRes.Entities)))
	}
	entity := entityLookupRes.Entities[0]

	buffer.Reset()

	providerName := determineProviderName(entity.Info.ShortTitle, entity.Info.FirstName, entity.Info.LastName)

	if err := supportMessageTemplate.Execute(&buffer, &messageContext{
		ProviderName: providerName,
	}); err != nil {
		return errors.Trace(err)
	}

	// Parse text and render as plain text so we can build a summary.
	textBML, err := bml.Parse(buffer.String())
	if e, ok := err.(bml.ErrParseFailure); ok {
		return errors.Trace(fmt.Errorf("failed to parse text at pos %d: %s", e.Offset, e.Reason))
	} else if err != nil {
		return errors.New("text is not valid markup")
	}
	plainText, err := textBML.PlainText()
	if err != nil {
		// Shouldn't fail here since the parsing should have done validation
		return errors.Trace(err)
	}
	summary := "Automated message from Spruce support"

	if _, err := w.threading.PostMessage(ctx, &threading.PostMessageRequest{
		ThreadID:     event.SpruceSupportThreadID,
		FromEntityID: res.Thread.PrimaryEntityID,
		Message: &threading.MessagePost{
			Text:    plainText,
			Summary: summary,
		},
	}); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func withinSupportHours(timeInPST time.Time) bool {

	if timeInPST.Hour() < spruceSupportStartHour {
		return false
	}

	if timeInPST.Hour() == spruceSupportStartHour {
		return timeInPST.Minute() >= spruceSupportStartMinute
	}

	if timeInPST.Hour() > spruceSupportEndHour {
		return false
	}

	if timeInPST.Hour() == spruceSupportEndHour {
		return timeInPST.Minute() <= spruceSupportEndMinute
	}

	return true
}

func determineProviderName(shortTile, firstName, lastName string) string {
	if _, ok := doctorTitles[shortTile]; ok {
		return fmt.Sprintf("Dr. %s", lastName)
	}
	return firstName
}
