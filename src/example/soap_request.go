package main

import (
	"carefront/libs/erx"
	"fmt"
)

func main() {
	medicationSearch := erx.NewMedicationQuickSearchMessage()
	medicationSearch.SSO = erx.GenerateSingleSignOn()
	medicationSearch.SearchString = "pro"

	searchResult := &erx.MedicationQuickSearchResult{}
	err := erx.MakeSoapRequest(medicationSearch, searchResult)

	if err != nil {
		panic(err.Error())
	}
	fmt.Println(searchResult.DisplayNames)
}
