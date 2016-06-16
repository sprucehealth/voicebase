package feedback

import "fmt"

// ForCase returns a feedback reason format for a patient case.
func ForCase(caseID int64) string {
	return fmt.Sprintf("case:%d", caseID)
}
