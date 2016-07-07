package manager

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

var (
	errPharmacyNotSet = errors.New("Please pick a pharmacy to continue.")
)

type pharmacyScreen struct {
	*screenInfo
}

func (p *pharmacyScreen) staticInfoCopy(context map[string]string) interface{} {
	return &pharmacyScreen{
		screenInfo: p.screenInfo.staticInfoCopy(nil).(*screenInfo),
	}
}

func (q *pharmacyScreen) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	var err error
	q.screenInfo, err = populateScreenInfo(data, parent)
	if err != nil {
		return err
	}

	// get the title from the parent if the parent has a title
	p, ok := parent.(titler)
	if ok {
		q.screenInfo.Title = p.title()
	}

	return nil
}

func (q *pharmacyScreen) TypeName() string {
	return screenTypePharmacy.String()
}

func (q *pharmacyScreen) transformToProtobuf() (proto.Message, error) {
	sInfo, err := transformScreenInfoToProtobuf(q.screenInfo)
	if err != nil {
		return nil, err
	}

	return &intake.PharmacyScreen{
		ScreenInfo: sInfo.(*intake.CommonScreenInfo),
	}, nil
}

func (s *pharmacyScreen) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	pharmacySet := dataSource.valueForKey(keyTypeIsPatientPharmacySet.String())
	pharmacySetBool, ok := pharmacySet.(bool)
	if !ok {
		return false, errPharmacyNotSet
	}
	if pharmacySetBool {
		return true, nil
	}

	return false, errPharmacyNotSet
}

func (s *pharmacyScreen) stringIndent(indent string, depth int) string {
	return fmt.Sprintf(indentAtDepth(indent, depth)+"%s: %s | %s", s.layoutUnitID(), screenTypePharmacy.String(), s.v)
}
