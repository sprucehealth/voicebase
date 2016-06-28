package manager

import (
	"fmt"
	"hash"
	"hash/fnv"
	"strconv"
)

type subscreenConfig struct {
	text    string
	screens []screen
}

// subquestionsManager works on a single question, and is responsible for:
// a) unmarshalling the subquestions config for a given question.
// b) creating deep copies of screens per patient answer item when the top level answer to a multiple choice or
// 	  autocomplete question is set. This includes setting the layoutUnitID for the subscreen, and unique IDs
// 	  for all questions contained within it.
type subquestionsManager struct {
	questionRef   question
	parentScreen  *questionScreen
	screenConfigs []screen
	h             hash.Hash32

	// dataSource needs to be held on to
	// so that screens and questions can be dynamically registered/deregistered
	// as they are added based on patient answers.
	dataSource    questionAnswerDataSource
	subScreensMap map[string]*subscreenConfig
}

func (s *subquestionsManager) staticInfoCopy(context map[string]string) interface{} {
	smCopy := &subquestionsManager{
		questionRef:   s.questionRef,
		screenConfigs: make([]screen, len(s.screenConfigs)),
	}

	copy(smCopy.screenConfigs, s.screenConfigs)
	return smCopy
}

func newSubquestionsManagerForQuestion(q question, dataSource questionAnswerDataSource) *subquestionsManager {
	return &subquestionsManager{
		questionRef:   q,
		parentScreen:  q.layoutParent().(*questionScreen),
		h:             fnv.New32a(),
		dataSource:    dataSource,
		subScreensMap: make(map[string]*subscreenConfig),
	}
}

func (s *subquestionsManager) unmarshalMapFromClient(data dataMap) error {
	if err := data.requiredKeys("subquestions_config", "screens"); err != nil {
		return err
	}

	screens, err := data.getInterfaceSlice("screens")
	if err != nil {
		return err
	}

	s.screenConfigs = make([]screen, len(screens))
	for i, sItem := range screens {
		screenMap, err := getDataMap(sItem)
		if err != nil {
			return err
		}

		s.screenConfigs[i], err = getScreen(screenMap, s.parentScreen, s.dataSource)
		if err != nil {
			return err
		}
	}

	// once the datasource setup is complete, inflate subscreens
	// for any existing top level answers. Note the reason we wait until
	// the datasource setup is complete is because the screen inflation
	// expects layoutUnitIDs to be assigned to all nodes, which is expected
	// to be the case after the datasource has been completely setup.
	if err := s.dataSource.subscribeForSetupCompletionNotification(func() error {
		if err := s.inflateSubscreensForPatientAnswer(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (s *subquestionsManager) computeIDForAnswerItem(item topLevelAnswerItem) string {
	s.h.Reset()
	answerID := make([]byte, 0, len(item.potentialAnswerID())+1+len(item.text()))
	if len(item.potentialAnswerID()) > 0 {
		answerID = append(answerID, []byte(item.potentialAnswerID())...)
	}

	answerID = append(answerID, []byte("|")...)
	answerID = append(answerID, []byte(item.text())...)
	s.h.Write(answerID)
	return strconv.FormatInt(int64(s.h.Sum32()), 10)
}

func (s *subquestionsManager) subscreensForAnswer(tItem topLevelAnswerItem) []screen {
	if sConfig := s.subScreensMap[s.computeIDForAnswerItem(tItem)]; sConfig != nil {
		return sConfig.screens
	}
	return nil
}

// inflateSubscreensForPatientAnswer creates the appropriate subscreens
// for each patient answer item based on the patient answer.
// Note that this method assumes that the datasource has been setup already.
func (s *subquestionsManager) inflateSubscreensForPatientAnswer() error {

	if err := s.deRegisterAllSubscreens(); err != nil {
		return err
	}
	pa, err := s.questionRef.patientAnswer()
	if err == errNoAnswerExists || pa.isEmpty() {
		return nil
	}

	topLevelAnswersContainer, ok := pa.(topLevelAnswerWithSubScreensContainer)
	if !ok {
		return fmt.Errorf("Patient answer (%T) expected to be a container of top level answers that potentially had subanswers but not the case", pa)
	}

	topLevelAnswers := topLevelAnswersContainer.topLevelAnswers()
	subscreens := make([]screen, 0, len(topLevelAnswers)*len(s.screenConfigs))
	for _, item := range topLevelAnswers {

		// compute a layoutUnitID specifically for this answer
		hash := s.computeIDForAnswerItem(item)
		uniqueIDForAnswer := s.questionRef.layoutUnitID() + "|" + subquestionDescriptor + ":" + hash + "|"

		var sConfig *subscreenConfig
		if len(item.subscreens()) > 0 {
			sConfig = &subscreenConfig{
				text:    item.text(),
				screens: item.subscreens(),
			}
		} else {
			sConfig, err = s.createSubscreenConfigForAnswer(
				item,
				uniqueIDForAnswer,
				item.subAnswers())
			if err != nil {
				return err
			}
			// set the inflated subscreens at the answer item level
			item.setSubscreens(sConfig.screens)
		}

		subscreens = append(subscreens, sConfig.screens...)
		s.subScreensMap[hash] = sConfig
	}

	return s.dataSource.registerSubscreensForQuestion(s.questionRef, subscreens)
}

// createSubscreenForAnswer manages the inflation and populating of subscreens for each of the individual answers
// and maps any provided subanswers to the appropriate questions on the subscreens.
func (s *subquestionsManager) createSubscreenConfigForAnswer(item topLevelAnswerItem, layoutUnitIDPrefix string, subanswers []patientAnswer) (*subscreenConfig, error) {
	sConfig := &subscreenConfig{
		text:    item.text(),
		screens: make([]screen, len(s.screenConfigs)),
	}

	// create a map of questionID -> answer so that we can easily map answers to subquestions
	answerMap := make(map[string]patientAnswer, len(subanswers))
	for _, pa := range subanswers {
		answerMap[pa.questionID()] = pa
	}

	for i, sItem := range s.screenConfigs {
		sConfig.screens[i] = sItem.staticInfoCopy(map[string]string{
			"answer": item.text(),
		}).(screen)

		sConfig.screens[i].setLayoutParent(s.parentScreen.layoutParent())

		// add a condition to each subscreen to depend on the top level answer
		// item.
		cond := sConfig.screens[i].condition()
		topLevelAnswerCondition := &answerContainsAnyCondition{
			answerCondition: answerCondition{
				Op:                 conditionTypeAnswerContainsAny.String(),
				QuestionID:         s.questionRef.id(),
				PotentialAnswersID: []string{item.potentialAnswerID()},
			},
		}
		if cond != nil {
			cond = &andCondition{
				logicalCondition: logicalCondition{
					Op: conditionTypeAND.String(),
					Operands: []condition{
						cond,
						topLevelAnswerCondition,
					},
				},
			}
		} else {
			cond = topLevelAnswerCondition
		}
		sConfig.screens[i].setCondition(cond)

		// set a unique layout unit id for each instance of the screen for each answer selection
		setLayoutUnitIDForNode(sConfig.screens[i], i, layoutUnitIDPrefix)

		qContainer, ok := sConfig.screens[i].(questionsContainer)
		if ok {
			for _, qItem := range qContainer.questions() {

				// populate the answer for the question
				// if it so exists
				pa, ok := answerMap[qItem.id()]
				if ok {
					if err := qItem.setPatientAnswer(pa); err != nil {
						return nil, err
					}
				}

				// update the questionID to be unique for each instance of the question
				// for each answer selection. the reason we do this is so that
				// when the client sets the answer to a subquestion, we know exactly
				// which top level question to map given that the subquestion is unique to the
				// particular top level question.
				qItem.setID(qItem.layoutUnitID() + "_" + qItem.id())

				// set the parent question to be the top level question as that will be useful
				// information in determining who to request for the marshalling of the top
				// level answer to the question.
				qItem.setParentQuestion(s.questionRef)
			}
		}
	}

	return sConfig, nil
}

func (s *subquestionsManager) deRegisterAllSubscreens() error {

	if err := s.dataSource.deregisterSubscreensForQuestion(s.questionRef); err != nil {
		return err
	}

	// deregister all screens from the datasource
	// and empty out map
	for answerID := range s.subScreensMap {
		delete(s.subScreensMap, answerID)
	}

	return nil
}
