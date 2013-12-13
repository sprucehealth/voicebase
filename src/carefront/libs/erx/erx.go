package erx

type ERxAPI interface {
	GetDrugNames(prefix string) ([]string, error)
}
