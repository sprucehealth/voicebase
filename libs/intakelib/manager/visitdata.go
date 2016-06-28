package manager

import (
	"encoding/json"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type platform string

const (
	android platform = "android"
	ios     platform = "ios"
)

type visitData struct {
	patientVisitID int64
	isSubmitted    bool
	layoutData     dataMap
	userFields     *userFields
	platform       platform
}

// unmarshal parses the incoming data into the visitData by using dataType
// to determine how to parse the incoming data.
func (v *visitData) unmarshal(dataType string, data []byte) error {
	var vd intake.VisitData
	if err := proto.Unmarshal(data, &vd); err != nil {
		return err
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(vd.Layout, &jsonMap); err != nil {
		return err
	}

	v.patientVisitID = *vd.PatientVisitId
	v.layoutData = jsonMap
	v.isSubmitted = *vd.IsSubmitted
	v.userFields = &userFields{}
	for _, pair := range vd.Pairs {
		if err := v.userFields.set(*pair.Key, pair.Value); err != nil {
			return err
		}
	}

	switch *vd.Platform {
	case intake.VisitData_ANDROID:
		v.platform = android
	case intake.VisitData_IOS:
		v.platform = ios
	}

	return nil
}
