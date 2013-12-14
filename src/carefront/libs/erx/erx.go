package erx

type ERxAPI interface {
	GetDrugNamesForDoctor(prefix string) ([]string, error)
	GetDrugNamesForPatient(prefix string) ([]string, error)
}
