package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/test"
)

func TestIsValidPlane0Unicode(t *testing.T) {
	test.Equals(t, true, isValidPlane0Unicode(`This is a välid string`))
	test.Equals(t, false, isValidPlane0Unicode(`This is not 😡`))
}

func TestInitialsForEntity(t *testing.T) {
	test.Equals(t, "", initialsForEntity(&models.Entity{FirstName: "", LastName: ""}))
	test.Equals(t, "A", initialsForEntity(&models.Entity{FirstName: "Aphex", LastName: ""}))
	test.Equals(t, "Z", initialsForEntity(&models.Entity{FirstName: "", LastName: "Zappa"}))
	test.Equals(t, "AZ", initialsForEntity(&models.Entity{FirstName: "Aphex", LastName: "Zappa"}))
	test.Equals(t, "👀Ž", initialsForEntity(&models.Entity{FirstName: "👀phex", LastName: "Žappa"}))
}
