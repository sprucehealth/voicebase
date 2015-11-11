package app_worker

import (
	"math/rand"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/errors"
)

func identifyProvider(dataAPI api.DataAPI, doctor *common.Doctor) (*common.Doctor, string, error) {
	cc, err := dataAPI.ListCareProviders(api.LCPOptPrimaryCCOnly)
	if err != nil {
		return nil, "", errors.Trace(err)
	}

	if len(cc) == 0 {
		return doctor, api.RoleDoctor, nil
	}

	return cc[rand.Intn(len(cc))], api.RoleCC, nil
}
