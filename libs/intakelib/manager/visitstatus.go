package manager

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/cmd/svc/restapi/app_url"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

// completionStatusType represents the completion status of a particular component.
// Zero value is statusTypeUncomputed.
type completionStatusType int

const (
	// statusTypeUncomputed represents a completion status that has not yet been determined.
	statusTypeUncomputed completionStatusType = iota

	// statusTypeIncomplete represents an incomplete completion status for a component (eg a section does
	// not yet have its requirements met so its not complete for a user.)
	statusTypeIncomplete

	// statusTypeComplete represents a complete completion status for a component.
	statusTypeComplete
)

// sectionCompletionStatus represents the overall completion status and state
// of a single section within the visit.
type sectionCompletionStatus struct {

	// sectionID represents the ID for which the completion status
	// is being calculated.
	sectionID string

	// lastShownStatus represents the completion status as shown in the last
	// representation of the visit overview screen.
	lastShownStatus completionStatusType

	// currentStatus represents the up-to-date completion status
	// of the section.
	currentStatus completionStatusType

	// resumeScreenID represents the screenID from which the client should resume a particular
	// section.
	resumeScreenID string
}

// visitCompletionStatus represents the completion status
// of all sections contained within a visit. Note that
// the sections are represented in order in which they occur in the visit.
type visitCompletionStatus struct {
	statuses     []*sectionCompletionStatus
	visitManager *visitManager

	// resumeScreenID points to the screen to resume the visit on
	// after the user is shown the visit overview screen.
	// if all requirements for the visit have been met, resumeScreenID is an empty string.
	resumeScreenID string

	// resumeSectionIndex points the section to resume the visit on
	// after the user is shown the visit overview screen.
	// if all requirements for the visit have been met, resumeSectionIndex
	// points to the index after the last section index.
	resumeSectionIndex int
}

// newVisitCompletionStatus returns a new object that represents the overall status of the visit.
func newVisitCompletionStatus(vm *visitManager) (*visitCompletionStatus, error) {
	vs := &visitCompletionStatus{
		visitManager: vm,
	}

	vs.statuses = make([]*sectionCompletionStatus, len(vm.visit.Sections))
	for i, section := range vm.visit.Sections {
		vs.statuses[i] = &sectionCompletionStatus{
			sectionID: section.layoutUnitID(),
		}
	}

	return vs, vs.update()
}

// update updates the overall visit completion status based on the contents of the datasource.
func (vs *visitCompletionStatus) update() error {

	var incompleteSectionSeen bool
	for i, section := range vs.visitManager.visit.Sections {
		// recompute the resumeScreenID for a section with incoming information in the datasource.
		resumeScreen, err := vs.visitManager.computeNextScreenInSection(section, nil)
		if err != nil {
			return err
		} else if resumeScreen == nil {
			return fmt.Errorf("Unable to find a screen to resume section %s from", section.layoutUnitID())
		}
		vs.statuses[i].resumeScreenID = resumeScreen.layoutUnitID()

		res, err := section.requirementsMet(vs.visitManager)
		if !res || err != nil {
			vs.statuses[i].currentStatus = statusTypeIncomplete

			if !incompleteSectionSeen {
				vs.resumeScreenID = vs.statuses[i].resumeScreenID
				vs.resumeSectionIndex = i
				incompleteSectionSeen = true
			}
		} else {
			vs.statuses[i].currentStatus = statusTypeComplete
		}
	}

	if !incompleteSectionSeen {
		vs.resumeScreenID = ""
		vs.resumeSectionIndex = len(vs.statuses)
	}

	return nil
}

func (vs *visitCompletionStatus) updateLastShownStatuses() {
	for _, status := range vs.statuses {
		status.lastShownStatus = status.currentStatus
	}
}

func (vs *visitCompletionStatus) transformToProtobuf() (proto.Message, error) {
	status := intake.VisitStatus{
		Entries: make([]*intake.VisitStatus_StatusEntry, len(vs.statuses)),
	}

	for i, completionStatus := range vs.statuses {

		var state *intake.VisitStatus_StatusEntry_State
		switch completionStatus.currentStatus {
		case statusTypeComplete:
			state = intake.VisitStatus_StatusEntry_COMPLETE.Enum()
		case statusTypeIncomplete, statusTypeUncomputed:
			state = intake.VisitStatus_StatusEntry_INCOMPLETE.Enum()
		}

		status.Entries[i] = &intake.VisitStatus_StatusEntry{
			Name:    proto.String(vs.visitManager.visit.Sections[i].Title),
			TapLink: proto.String(app_url.ViewVisitScreen(completionStatus.resumeScreenID).String()),
			State:   state,
		}
	}

	return &status, nil
}
