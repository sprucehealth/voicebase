package manager

import (
	"bytes"
	"fmt"
	"runtime"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

var (
	errVisitReadOnlyMode = errors.New("cannot modify visit in read only mode")
)

type transitionItem struct {
	Message string    `json:"message"`
	Buttons []*button `json:"buttons"`
}

func (v *transitionItem) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("transition", "message", "buttons"); err != nil {
		return errors.Trace(err)
	}

	v.Message = data.mustGetString("message")
	buttons, err := data.getInterfaceSlice("buttons")
	if err != nil {
		return errors.Trace(err)
	}

	v.Buttons = make([]*button, len(buttons))
	for i, bItem := range buttons {
		buttonMap, err := getDataMap(bItem)
		if err != nil {
			return errors.Trace(err)
		}

		v.Buttons[i], err = populateButton(buttonMap)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

type visitOverviewHeader struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	IconURL  string `json:"icon_url"`
}

func (v *visitOverviewHeader) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("header", "title"); err != nil {
		return err
	}

	v.Title = data.mustGetString("title")
	v.Subtitle = data.mustGetString("subtitle")

	if data.exists("icon_url") {
		v.IconURL = data.mustGetString("icon_url")
	}

	return nil
}

type visit struct {
	LayoutUnitID   string               `json:"-"`
	Sections       []*section           `json:"sections"`
	Transitions    []*transitionItem    `json:"transitions"`
	OverviewHeader *visitOverviewHeader `json:"header"`
	IsSubmitted    bool                 `json:"is_submitted"`
	ID             string               `json:"id"`

	v      visibility
	parent layoutUnit

	// TODO transitions
}

func (v *visit) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) (err error) {

	defer func() {
		if rerr := recover(); rerr != nil {
			if _, ok := rerr.(runtime.Error); ok {
				panic(err)
			}

			err = fmt.Errorf("Unable to unmarshalMap object into visit: %v", rerr.(error))
		}
	}()

	if err := data.requiredKeys("intake_container", "header", "intake"); err != nil {
		return errors.Trace(err)
	}

	intake, err := getDataMap(data.get("intake"))
	if err != nil {
		return errors.Trace(err)
	}

	if err := intake.requiredKeys("intake", "sections", "transitions"); err != nil {
		return errors.Trace(err)
	}

	sections, err := intake.getInterfaceSlice("sections")
	if err != nil {
		return errors.Trace(err)
	}

	v.Sections = make([]*section, len(sections))
	for i, sectionVal := range sections {
		sectionMap, err := getDataMap(sectionVal)
		if err != nil {
			return err
		}

		sItem := &section{}
		if err := sItem.unmarshalMapFromClient(sectionMap, v, dataSource); err != nil {
			return errors.Trace(err)
		}

		v.Sections[i] = sItem
	}

	transitions, err := intake.getInterfaceSlice("transitions")
	if err != nil {
		return err
	}

	v.Transitions = make([]*transitionItem, len(transitions))
	for i, tItem := range transitions {
		transitionMap, err := getDataMap(tItem)
		if err != nil {
			return errors.Trace(err)
		}

		v.Transitions[i] = &transitionItem{}
		if err := v.Transitions[i].unmarshalMapFromClient(transitionMap); err != nil {
			return errors.Trace(err)
		}
	}

	// enforce that there are N+1 transition items where N = number of sections
	if len(v.Transitions) != len(v.Sections)+1 {
		return fmt.Errorf("Malformed visit. Expect %d transition items for %d sections. Instead, got %d transition items.",
			len(v.Sections)+1, len(v.Sections), len(v.Transitions))
	}

	overviewMap, err := data.dataMapForKey("header")
	if err != nil {
		return errors.Trace(err)
	}

	v.OverviewHeader = &visitOverviewHeader{}
	if err := v.OverviewHeader.unmarshalMapFromClient(overviewMap); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (v *visit) condition() condition {
	return nil
}

func (v *visit) setCondition(cond condition) {}

func (v *visit) descriptor() string {
	return "v"
}

func (v *visit) layoutParent() layoutUnit {
	return nil
}

func (v *visit) setLayoutParent(node layoutUnit) {
	v.parent = node
}

func (v *visit) children() []layoutUnit {
	children := make([]layoutUnit, len(v.Sections))
	for i, sItem := range v.Sections {
		children[i] = sItem
	}

	return children
}

func (v *visit) setLayoutUnitID(str string) {
	v.LayoutUnitID = str
}

func (v *visit) layoutUnitID() string {
	return v.LayoutUnitID
}

func (v *visit) setLayoutUnitIDsForSections() {
	for i, section := range v.Sections {
		setLayoutUnitIDForNode(section, i, "")
	}
}

func (vt *visit) setVisibility(vi visibility) {
	vt.v = vi
}

func (vt *visit) visibility() visibility {
	return vt.v
}

// requirementsMet returns no error if a visit has all its requirements met
func (v *visit) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	for _, se := range v.Sections {
		if res, err := se.requirementsMet(dataSource); err != nil {
			return res, err
		} else if !res {
			return false, nil
		}
	}

	return true, nil
}

// questions returns all the questions contained within
// all question screens within the layout.
func (v *visit) questions() []question {
	var questions []question
	for _, section := range v.Sections {
		for _, screen := range section.Screens {
			qContainer, ok := screen.(questionsContainer)
			if ok {
				questions = append(questions, qContainer.questions()...)
			}
		}
	}

	return questions
}

// validateVisit wraps any validation errors in a custom object to be sent to the client to parse out
// and appropriately display validation errors. If there are no validation errors, then that is communicated
// via the data object as well.
func (v *visit) validateRequirements(dataSource questionAnswerDataSource, serializerLib serializer) ([]byte, error) {
	var res intake.ValidateRequirementsResult

	requirementsMet, err := v.requirementsMet(dataSource)
	if err != nil {
		res.Status = intake.ValidateRequirementsResult_ERROR.Enum()
		res.Message = proto.String(err.Error())
	} else if !requirementsMet {
		res.Status = intake.ValidateRequirementsResult_ERROR.Enum()
	} else {
		res.Status = intake.ValidateRequirementsResult_OK.Enum()
	}

	return serializerLib.marshal(&res)
}

func (v *visit) String() string {
	var b bytes.Buffer
	for _, section := range v.Sections {
		for _, screen := range section.Screens {
			b.WriteString(screen.stringIndent("  ", 0))
			b.WriteString("\n\n")
		}
	}

	return b.String()
}
