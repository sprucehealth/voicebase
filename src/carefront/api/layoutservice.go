package api

import (
	"bytes"
	"fmt"
)

type LayoutService struct {
	DataAPI        *DataService
	CloudObjectAPI *CloudObjectService
}

func (l *LayoutService) VerifyAndUploadIncomingLayout(rawLayout []byte, treatmentTag string) error {
	fmt.Println("Verifying layout for ", treatmentTag)
	currentActiveBucket, currentActiveKey, currentActiveRegion, _ := l.DataAPI.GetCurrentActiveLayoutInfoForTreatment(treatmentTag)
	rawData, err := l.CloudObjectAPI.GetObjectAtLocation(currentActiveBucket, currentActiveKey, currentActiveRegion)
	if err != nil {
		return err
	}

	res := bytes.Compare(rawLayout, rawData)
	fmt.Println(res)
	return nil
}
