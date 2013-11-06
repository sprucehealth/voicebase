package api

import (
	"fmt"
)

type LayoutService struct {
	DataAPI *DataService
}

func (l *LayoutService) VerifyAndUploadIncomingLayout(rawLayout []byte, treatmentTag string) error {
	fmt.Println("Verifying layout for ", treatmentTag)
	currentActiveBucket, currentActiveKey, currentActiveRegion, _ := l.DataAPI.GetCurrentActiveLayoutInfoForTreatment(treatmentTag)
	fmt.Println(currentActiveBucket, currentActiveKey, currentActiveRegion)
	return nil
}
