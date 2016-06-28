package manager

import (
	"container/list"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/restapi/app_url"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/test"
)

type mockDataSource_visitStatus struct {
	q question
	questionAnswerDataSource
}

func (m *mockDataSource_visitStatus) question(questionID string) question {
	return m.q
}

func TestVisitStatus(t *testing.T) {
	v := &visit{
		Sections: []*section{
			{
				LayoutUnitID: "se1",
				Screens: []screen{
					&questionScreen{
						screenInfo: &screenInfo{},
						Questions: []question{
							&multipleChoiceQuestion{
								questionInfo: &questionInfo{},
							},
						},
					},
				},
			},
		},
	}

	l := list.New()
	l.PushBack(v.Sections[0].Screens[0])

	vs, err := newVisitCompletionStatus(&visitManager{
		visit: v,
		sectionScreensMap: map[string]*list.List{
			"se1": l,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 1, len(vs.statuses))
	test.Equals(t, statusTypeComplete, vs.statuses[0].currentStatus)

	test.OK(t, vs.update())

	test.Equals(t, statusTypeComplete, vs.statuses[0].currentStatus)
	test.Equals(t, statusTypeUncomputed, vs.statuses[0].lastShownStatus)

	// now lets update the only question to make it required
	v.Sections[0].Screens[0].(*questionScreen).Questions[0].(*multipleChoiceQuestion).Required = true

	test.OK(t, vs.update())

	// at this point the status should transition from complete -> incomplete
	test.Equals(t, statusTypeIncomplete, vs.statuses[0].currentStatus)
}

func TestVisitStatus_resumeScreenID(t *testing.T) {
	v := &visit{
		Sections: []*section{
			{
				LayoutUnitID: "se1",
				Screens: []screen{
					&questionScreen{
						screenInfo: &screenInfo{
							LayoutUnitID: "se:0|sc:0",
						},
						Questions: []question{
							&multipleChoiceQuestion{
								questionInfo: &questionInfo{},
							},
						},
					},
				},
			},
			{
				LayoutUnitID: "se2",
				Screens: []screen{
					&questionScreen{
						screenInfo: &screenInfo{
							LayoutUnitID: "se:1|sc:0",
							v:            hidden,
						},
						Questions: []question{
							&multipleChoiceQuestion{
								questionInfo: &questionInfo{
									Required: true,
								},
							},
						},
					},
					&questionScreen{
						screenInfo: &screenInfo{
							LayoutUnitID: "se:1|sc:1",
						},
						Questions: []question{
							&multipleChoiceQuestion{
								questionInfo: &questionInfo{
									Required: true,
								},
							},
						},
					},
				},
			},
		},
	}

	l1 := list.New()
	l1.PushBack(v.Sections[0].Screens[0])

	l2 := list.New()
	l2.PushBack(v.Sections[1].Screens[0])
	l2.PushBack(v.Sections[1].Screens[1])

	vs, err := newVisitCompletionStatus(&visitManager{
		visit: v,
		sectionScreensMap: map[string]*list.List{
			"se1": l1,
			"se2": l2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	test.OK(t, vs.update())

	// would expect the resumeScreenID to be the first screen of the second section
	test.Equals(t, "se:1|sc:1", vs.resumeScreenID)
	test.Equals(t, 1, vs.resumeSectionIndex)

	// for the second section would expect it to be second screen given first screen has a condition that is not met
	test.Equals(t, "se:1|sc:1", vs.statuses[1].resumeScreenID)

	// now go ahead and mark the required question as not required so that all requirements of the visit
	// are completed.
	v.Sections[1].Screens[1].(*questionScreen).Questions[0].(*multipleChoiceQuestion).Required = false

	test.OK(t, vs.update())

	// there should be no screen to resume the visit from given that all requirements are met.
	test.Equals(t, "", vs.resumeScreenID)
	test.Equals(t, 2, vs.resumeSectionIndex)

}

func TestVisitStatus_transform(t *testing.T) {
	v := &visit{
		Sections: []*section{
			{
				LayoutUnitID: "se1",
				Title:        "Testing",
				Screens: []screen{
					&questionScreen{
						screenInfo: &screenInfo{
							LayoutUnitID: "se:0|sc:0",
						},
						Questions: []question{
							&multipleChoiceQuestion{
								questionInfo: &questionInfo{},
							},
						},
					},
				},
			},
			{
				LayoutUnitID: "se2",
				Title:        "Testing2",
				Screens: []screen{
					&questionScreen{
						screenInfo: &screenInfo{
							LayoutUnitID: "se:1|sc:0",
						},
						Questions: []question{
							&multipleChoiceQuestion{
								questionInfo: &questionInfo{},
							},
						},
					},
				},
			},
		},
	}

	l1 := list.New()
	l1.PushBack(v.Sections[0].Screens[0])

	l2 := list.New()
	l2.PushBack(v.Sections[1].Screens[0])

	vs, err := newVisitCompletionStatus(&visitManager{
		visit: v,
		sectionScreensMap: map[string]*list.List{
			"se1": l1,
			"se2": l2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	pb, err := vs.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	status, ok := pb.(*intake.VisitStatus)
	test.Equals(t, true, ok)
	test.Equals(t, 2, len(status.Entries))
	test.Equals(t, "Testing", *status.Entries[0].Name)
	test.Equals(t, intake.VisitStatus_StatusEntry_COMPLETE, *status.Entries[0].State)
	test.Equals(t, app_url.ViewVisitScreen("se:0|sc:0").String(), *status.Entries[0].TapLink)
	test.Equals(t, "Testing2", *status.Entries[1].Name)
	test.Equals(t, intake.VisitStatus_StatusEntry_COMPLETE, *status.Entries[1].State)
	test.Equals(t, app_url.ViewVisitScreen("se:1|sc:0").String(), *status.Entries[1].TapLink)

	// now lets update the only question to make it required
	v.Sections[0].Screens[0].(*questionScreen).Questions[0].(*multipleChoiceQuestion).Required = true

	test.OK(t, vs.update())

	pb, err = vs.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}
	status = pb.(*intake.VisitStatus)

	test.Equals(t, intake.VisitStatus_StatusEntry_INCOMPLETE, *status.Entries[0].State)
	test.Equals(t, app_url.ViewVisitScreen("se:0|sc:0").String(), *status.Entries[0].TapLink)
	test.Equals(t, intake.VisitStatus_StatusEntry_COMPLETE, *status.Entries[1].State)
	test.Equals(t, app_url.ViewVisitScreen("se:1|sc:0").String(), *status.Entries[1].TapLink)
}
