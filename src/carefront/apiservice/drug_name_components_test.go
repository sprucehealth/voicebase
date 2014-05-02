package apiservice

import (
	"fmt"
	"testing"
)

func TestBreakingUpOfDrugNameIntoComponents(t *testing.T) {
	expectedDrugNameComponent := "DrugName"
	expectedDrugRouteComponent := "DrugRoute"
	expectedDrugFormComponent := "DrugForm"
	nameToTest := fmt.Sprintf("%s (%s - %s)", expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent)
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)

	expectedDrugNameComponent = "DrugName"
	expectedDrugRouteComponent = "DrugRoute"
	expectedDrugFormComponent = "DrugForm -"
	nameToTest = fmt.Sprintf("%s (%s - %s)", expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent)
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)

	expectedDrugNameComponent = "DrugName"
	expectedDrugRouteComponent = "DrugRoute"
	expectedDrugFormComponent = "DrugForm -"
	nameToTest = fmt.Sprintf("  %s   (    %s -      %s   )", expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent)
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)
}

func TestUnsuccessfulBreakingUpOfDrugNameIntoComponents(t *testing.T) {
	expectedDrugNameComponent := ""
	expectedDrugRouteComponent := ""
	expectedDrugFormComponent := ""
	nameToTest := ""
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)

	expectedDrugNameComponent = "!2315151"
	expectedDrugRouteComponent = ""
	expectedDrugFormComponent = ""
	nameToTest = expectedDrugNameComponent
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, expectedDrugNameComponent, t)

	expectedDrugNameComponent = "!2315151 ( )"
	expectedDrugRouteComponent = ""
	expectedDrugFormComponent = ""
	nameToTest = expectedDrugNameComponent
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)

	expectedDrugNameComponent = "!2315151 ( -"
	expectedDrugRouteComponent = ""
	expectedDrugFormComponent = ""
	nameToTest = expectedDrugNameComponent
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)

	expectedDrugNameComponent = "!2315151"
	expectedDrugRouteComponent = ""
	expectedDrugFormComponent = ""
	nameToTest = expectedDrugNameComponent
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)

	expectedDrugNameComponent = "!2315151 ( 1241415 -"
	expectedDrugRouteComponent = ""
	expectedDrugFormComponent = ""
	nameToTest = expectedDrugNameComponent
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)

	expectedDrugNameComponent = "!2315151 ( 1241415 ) -"
	expectedDrugRouteComponent = ""
	expectedDrugFormComponent = ""
	nameToTest = expectedDrugNameComponent
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)

	expectedDrugNameComponent = "!2315151 ( 1241415 - 1241414"
	expectedDrugRouteComponent = ""
	expectedDrugFormComponent = ""
	nameToTest = expectedDrugNameComponent
	testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest, t)
}

func testDrugNameIntoComponents(expectedDrugNameComponent, expectedDrugRouteComponent, expectedDrugFormComponent, nameToTest string, t *testing.T) {
	drugName, drugForm, drugRoute := BreakDrugInternalNameIntoComponents(nameToTest)
	checkComponent(expectedDrugNameComponent, drugName, t)
	checkComponent(expectedDrugFormComponent, drugForm, t)
	checkComponent(expectedDrugRouteComponent, drugRoute, t)
}

func checkComponent(expectedResult, outputtedResult string, t *testing.T) {
	if outputtedResult != expectedResult {
		t.Fatalf("Expected component to be %s instead it was %s", expectedResult, outputtedResult)
	}
}
