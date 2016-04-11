package support

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	directorymock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/operational"
	"github.com/sprucehealth/backend/svc/threading"
	threadingmock "github.com/sprucehealth/backend/svc/threading/mock"
	"github.com/sprucehealth/backend/test"
)

const (
	timeFormat = "Jan 2, 2006 at 3:04pm"
)

func TestPostSupportMessage_DuringSupportHours_Wait(t *testing.T) {
	orgCreatedTime, err := time.ParseInLocation(timeFormat, "Jan 2, 2016 at 3:04pm", californiaLocation)
	test.OK(t, err)
	mclock := clock.NewManaged(orgCreatedTime)
	testWaitToPost(t, mclock, orgCreatedTime.Unix())
}

func TestPostSupportMessage_DuringSupportHours_Post(t *testing.T) {
	orgCreatedTime, err := time.ParseInLocation(timeFormat, "Jan 2, 2016 at 3:04pm", californiaLocation)
	test.OK(t, err)
	mclock := clock.NewManaged(orgCreatedTime.Add(time.Hour + postMessageThreshold))
	testSuccessfulPost(t, mclock, orgCreatedTime.Unix())

	orgCreatedTime, err = time.ParseInLocation(timeFormat, "Jan 2, 2016 at 10:29pm", californiaLocation)
	test.OK(t, err)
	mclock = clock.NewManaged(orgCreatedTime.Add(time.Hour + postMessageThreshold))
	testSuccessfulPost(t, mclock, orgCreatedTime.Unix())

	orgCreatedTime, err = time.ParseInLocation(timeFormat, "Jan 2, 2016 at 7:31am", californiaLocation)
	test.OK(t, err)
	mclock = clock.NewManaged(orgCreatedTime.Add(time.Hour + postMessageThreshold))
	testSuccessfulPost(t, mclock, orgCreatedTime.Unix())
}

func TestPostSupportMessage_AfterBusinessHours_Wait(t *testing.T) {
	orgCreatedTime, err := time.ParseInLocation(timeFormat, "Jan 2, 2016 at 11:04pm", californiaLocation)
	test.OK(t, err)
	mclock := clock.NewManaged(orgCreatedTime.Add(time.Hour + postMessageThreshold))
	testWaitToPost(t, mclock, orgCreatedTime.Unix())

	orgCreatedTime, err = time.ParseInLocation(timeFormat, "Jan 2, 2016 at 05:04am", californiaLocation)
	test.OK(t, err)
	mclock = clock.NewManaged(orgCreatedTime.Add(time.Hour + postMessageThreshold))
	testWaitToPost(t, mclock, orgCreatedTime.Unix())

	orgCreatedTime, err = time.ParseInLocation(timeFormat, "Jan 2, 2016 at 10:31pm", californiaLocation)
	test.OK(t, err)
	mclock = clock.NewManaged(orgCreatedTime.Add(time.Hour + postMessageThreshold))
	testWaitToPost(t, mclock, orgCreatedTime.Unix())

	orgCreatedTime, err = time.ParseInLocation(timeFormat, "Jan 2, 2016 at 7:29am", californiaLocation)
	test.OK(t, err)
	mclock = clock.NewManaged(orgCreatedTime.Add(time.Hour + postMessageThreshold))
	testWaitToPost(t, mclock, orgCreatedTime.Unix())
}

func TestPostSupportMessage_AfterBusinessHours_Post(t *testing.T) {
	orgCreatedTime, err := time.ParseInLocation(timeFormat, "Jan 2, 2016 at 11:04pm", californiaLocation)
	test.OK(t, err)
	mclock := clock.NewManaged(orgCreatedTime.Add(12 * time.Hour))
	testSuccessfulPost(t, mclock, orgCreatedTime.Unix())
}

func TestPostSupportMessage_AlreadyPosted(t *testing.T) {
	orgCreatedTime, err := time.ParseInLocation(timeFormat, "Jan 2, 2016 at 3:04pm", californiaLocation)
	test.OK(t, err)
	mclock := clock.NewManaged(orgCreatedTime.Add(12 * time.Hour))

	providerEntityID := "p1"
	spruceSupportThreadID := "t1"
	orgSupportThreadID := "t2"
	primaryEntityID := "pe1"

	mdir := directorymock.New(t)
	defer mdir.Finish()

	mthreading := threadingmock.New(t)
	defer mthreading.Finish()

	w := &Worker{
		directory: mdir,
		threading: mthreading,
		clock:     mclock,
	}

	mthreading.Expect(mock.NewExpectation(mthreading.Thread, &threading.ThreadRequest{
		ThreadID: spruceSupportThreadID,
	}).WithReturns(&threading.ThreadResponse{
		Thread: &threading.Thread{
			MessageCount:    2,
			PrimaryEntityID: primaryEntityID,
		},
	}, nil))

	err = w.processEvent(&operational.NewOrgCreatedEvent{
		SpruceSupportThreadID:   spruceSupportThreadID,
		OrgSupportThreadID:      orgSupportThreadID,
		InitialProviderEntityID: providerEntityID,
		OrgCreated:              orgCreatedTime.In(time.UTC).Unix(),
	})
	test.OK(t, err)
}

func testWaitToPost(t *testing.T, mclock clock.Clock, orgCreationTime int64) {
	providerEntityID := "p1"
	spruceSupportThreadID := "t1"
	orgSupportThreadID := "t2"

	mdir := directorymock.New(t)
	defer mdir.Finish()

	mthreading := threadingmock.New(t)
	defer mthreading.Finish()

	w := &Worker{
		directory: mdir,
		threading: mthreading,
		clock:     mclock,
	}

	err := w.processEvent(&operational.NewOrgCreatedEvent{
		SpruceSupportThreadID:   spruceSupportThreadID,
		OrgSupportThreadID:      orgSupportThreadID,
		InitialProviderEntityID: providerEntityID,
		OrgCreated:              orgCreationTime,
	})
	_, ok := errors.Cause(err).(*awsutil.ErrDelayedRetry)
	test.Equals(t, true, ok)
}

func testSuccessfulPost(t *testing.T, mclock clock.Clock, orgCreationTime int64) {
	providerEntityID := "p1"
	spruceSupportThreadID := "t1"
	orgSupportThreadID := "t2"
	primaryEntityID := "pe1"

	mdir := directorymock.New(t)
	defer mdir.Finish()

	mthreading := threadingmock.New(t)
	defer mthreading.Finish()

	w := &Worker{
		directory: mdir,
		threading: mthreading,
		clock:     mclock,
	}

	mdir.Expect(mock.NewExpectation(mdir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: providerEntityID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   providerEntityID,
				Type: directory.EntityType_INTERNAL,
				Info: &directory.EntityInfo{
					ShortTitle: "MD",
					LastName:   "Jham",
				},
			},
		},
	}, nil))

	mthreading.Expect(mock.NewExpectation(mthreading.Thread, &threading.ThreadRequest{
		ThreadID: spruceSupportThreadID,
	}).WithReturns(&threading.ThreadResponse{
		Thread: &threading.Thread{
			MessageCount:    1,
			PrimaryEntityID: primaryEntityID,
		},
	}, nil))

	mthreading.Expect(mock.NewExpectation(mthreading.PostMessage, &threading.PostMessageRequest{
		Text: `Hi Dr. Jham - great to see you on here! My name is Caitrin (I’m a real person, promise). We only recently launched Spruce, so I’m checking in with everyone that signs up to make sure the product makes sense. Any questions so far?

BTW, we put together a brief tutorial, which you can access here: bit.ly/22VjkkX.`,
		Summary:      "Automated message from Spruce support",
		FromEntityID: primaryEntityID,
		ThreadID:     spruceSupportThreadID,
	}))

	err := w.processEvent(&operational.NewOrgCreatedEvent{
		SpruceSupportThreadID:   spruceSupportThreadID,
		OrgSupportThreadID:      orgSupportThreadID,
		InitialProviderEntityID: providerEntityID,
		OrgCreated:              orgCreationTime,
	})
	test.OK(t, err)
}

func TestDetermineProviderName(t *testing.T) {
	for title := range doctorTitles {
		test.Equals(t, "Dr. Schmoe", determineProviderName(title, "Joe", "Schmoe"))
	}

	for _, title := range []string{"EMT",
		"LPC",
		"LPN",
		"LVN",
		"MA",
		"MS",
		"MSW",
		"NP",
		"PA",
		"PT",
		"RD",
		"RN"} {
		test.Equals(t, "Joe", determineProviderName(title, "Joe", "Schmoe"))
	}

}
