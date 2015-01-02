package diagnosis

type Diagnosis struct {
	ID          string
	Code        string
	Description string
	Billable    bool
}

type API interface {
	DoCodesExist(codeIDs []string) (bool, []string, error)
	DiagnosisForCodeIDs(codeIDs []string) (map[string]*Diagnosis, error)
	SynonymsForDiagnoses(codeIDs []string) (map[string][]string, error)
	SearchDiagnosesByCode(query string, numResults int) ([]*Diagnosis, error)
	SearchDiagnoses(query string, numResults int) ([]*Diagnosis, error)
	FuzzyTextSearchDiagnoses(query string, numResults int) ([]*Diagnosis, error)
}
