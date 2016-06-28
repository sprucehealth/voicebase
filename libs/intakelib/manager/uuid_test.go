package manager

import (
	"strings"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestGenerateUUID(t *testing.T) {
	uuid, err := generateUUID()
	if err != nil {
		t.Fatal(err)
	} else if uuid == "" {
		t.Fatalf("Unable to generate UUID")
	}

	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Fatalf("Expected 5 parts to UUID but got %d", len(parts))
	}

	for i, part := range parts {
		switch i {
		case 0:
			test.Equals(t, 8, len(part))
		case 1, 2, 3:
			test.Equals(t, 4, len(part))
		case 4:
			test.Equals(t, 12, len(part))
		}
	}

}
