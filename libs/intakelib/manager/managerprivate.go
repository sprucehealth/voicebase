package manager

// This file contains private methods for the visitManager type.

import (
	"container/list"
	"encoding/json"
	"fmt"
)

func (v *visitManager) screen(layoutUnitID string) (screen, error) {
	node, ok := v.layoutUnitMap[layoutUnitID]
	if !ok {
		return nil, fmt.Errorf("screen %s doesn't exist", layoutUnitID)
	}

	s, ok := node.(screen)
	if !ok {
		return nil, fmt.Errorf("screen %s doesn't exist", layoutUnitID)
	}

	return s, nil
}

type clientJSONStructure struct {
	Answers      map[string]interface{} `json:"answers,omitempty"`
	ClearAnswers []string               `json:"clear_answers,omitempty"`
}

// persistAllDirtyQuestions goes through the questionMap and
// requests the client to persist the answer to questions with dirty
// answers. The method expects to the caller to have acquired write locks
// to update book-keeping as necessary.
func (v *visitManager) persistAllDirtyQuestions() error {

	for _, qData := range v.questionMap {

		if !qData.isAnswerDirty {
			continue
		}

		if !qData.questionRef.canPersistAnswer() {
			continue
		}

		// upload the parent answer if the subquestion is marked as being dirty
		q := qData.questionRef
		if q.parentQuestion() != nil {
			q = q.parentQuestion()
		}

		clientJSON := &clientJSONStructure{}

		if q.visibility() == hidden {
			clientJSON.ClearAnswers = []string{q.id()}
		} else {
			answerJSON, err := q.answerForClient()
			if err != nil {
				return err
			}
			clientJSON.Answers = map[string]interface{}{
				q.id(): answerJSON,
			}
		}

		jsonData, err := json.Marshal(clientJSON)
		if err != nil {
			return err
		}

		cd := &clientAnswerData{
			questionID:   q.id(),
			questionType: q.TypeName(),
			answerJSON:   jsonData,
		}

		data, err := cd.marshalProtobuf()
		if err != nil {
			return err
		}

		if err := v.cli.PersistAnswerForQuestion(data); err != nil {
			return err
		}

		qData.isAnswerDirty = false

		// mark the fact that we persisted the answer to a prefilled question
		if qData.questionRef.prefilled() {
			qData.prefilledAnswerPersisted = true
		}
	}
	return nil
}

// registerDependencies registers the provided node
// as a dependent of its dependencies, which are also layoutUnits.
func (v *visitManager) registerNode(node layoutUnit, dependencies []layoutUnit) {
	if v.layoutUnitMap == nil {
		v.layoutUnitMap = make(map[string]layoutUnit)
	}
	v.layoutUnitMap[node.layoutUnitID()] = node

	// if the node is a question, add it to the question map
	q, ok := node.(question)
	if ok {
		v.questionMap[q.id()] = &questionData{
			questionRef: q,
		}
	}

	if v.dependantsMap == nil {
		v.dependantsMap = make(map[string][]layoutUnit)
	}

	for _, lUnitItem := range dependencies {
		currentItems := v.dependantsMap[lUnitItem.layoutUnitID()]
		v.dependantsMap[lUnitItem.layoutUnitID()] = append(currentItems, node)
	}
}

func (v *visitManager) deregisterNode(node layoutUnit) {
	delete(v.layoutUnitMap, node.layoutUnitID())
	delete(v.questionMap, node.layoutUnitID())
	delete(v.dependantsMap, node.layoutUnitID())

	// NOTE: while we could also remove the node from any list of dependants
	// in the map that contains the node itself, not doing so to reduce spinning
	// cycles when the node having its state updated is going to affect nothing at all.
	// Holding it in memory might turn out to be a problem, in which case we can figure out
	// how best to get rid of the node.
}

// computeCurrentVisibilityState recursively walks through all layout nodes
// to compute the visibility of each node. It first computes the visibility of its children
// before computing its own visible state. Dependents are evaluated upon updating the visibility
// of each node.
func (v *visitManager) computeCurrentVisibilityState(node layoutUnit) {
	for _, child := range node.children() {
		v.computeCurrentVisibilityState(child)
	}
	newVisibility := computeLayoutVisibility(node, v)
	node.setVisibility(newVisibility)
	v.evaluateDependants(node)
}

// evaluateDependants computes the visibility of the dependants of the provided node
// and updates the visibility state if it has changed. If the visibility has changed, then
// it recursively evaluates the dependants of each of the dependants.
func (v *visitManager) evaluateDependants(node layoutUnit) {
	for _, item := range v.dependantsMap[node.layoutUnitID()] {

		currentVisibility := item.visibility()
		newVisibility := computeLayoutVisibility(item, v)

		// fmt.Printf("%s %s->%s\n", item.layoutUnitID(), currentVisibility, newVisibility)
		// update the visibility of the item and then evaluate its dependants
		if newVisibility != currentVisibility {
			item.setVisibility(newVisibility)

			// if the node is a question, and has a patient answer, mark the question as being dirty
			// so that we can persist the latest answer to the question after the changed visibility state.
			q, ok := item.(question)
			if ok {
				_, err := q.patientAnswer()
				if err == nil {
					v.questionMap[q.id()].isAnswerDirty = true
				}
			}
			v.evaluateDependants(item)
		}
	}
}

// setupStaticScreenForAllSections iterates through each section
// and adds all static screens mapped to by the section's layoutUnitID
// to sectionScreensMap.
func (v *visitManager) setupStaticScreenForAllSections() {
	v.sectionScreensMap = make(map[string]*list.List, len(v.visit.Sections))
	for _, section := range v.visit.Sections {
		v.sectionScreensMap[section.LayoutUnitID] = list.New()
		for _, screen := range section.Screens {
			v.sectionScreensMap[section.LayoutUnitID].PushBack(screen)
		}
	}
}

func (v *visitManager) computeNextScreenInSection(se *section, currentScreen screen) (screen, error) {
	screenList, ok := v.sectionScreensMap[se.layoutUnitID()]
	if !ok {
		return nil, fmt.Errorf("section %s doesn't exist", se.layoutUnitID())
	}

	// if current screen is not specified, return the first visible screen within the section
	screenFound := currentScreen == nil
	for e := screenList.Front(); e != nil; e = e.Next() {
		if !screenFound && e.Value.(screen).layoutUnitID() == currentScreen.layoutUnitID() {
			screenFound = true
			continue
		}

		if screenFound && e.Value.(screen).visibility() == visible {
			return e.Value.(screen), nil
		}
	}

	// no visible screen in section found
	return nil, nil
}

// computeProgress computes the progress within a given section by dividing the
// position of the screen within the list by the size of the list.
func (v *visitManager) computeProgress(s screen) (*float32, error) {
	se, err := parentSectionForScreen(s)
	if err != nil {
		return nil, err
	}

	screenList, ok := v.sectionScreensMap[se.layoutUnitID()]
	if !ok {
		return nil, fmt.Errorf("section %s doesn't exist", se.layoutUnitID())
	}

	i := float32(0)
	for e := screenList.Front(); e != nil; e = e.Next() {
		if e.Value.(screen).layoutUnitID() == s.layoutUnitID() {
			progress := float32((i + 1.0) / float32(screenList.Len()))
			return &progress, nil
		}
		i++
	}

	return nil, fmt.Errorf("screen %s not found ", s.layoutUnitID())
}
