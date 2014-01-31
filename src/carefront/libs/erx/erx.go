package erx

type ERxAPI interface {
	GetDrugNamesForDoctor(prefix string) ([]string, error)
	GetDrugNamesForPatient(prefix string) ([]string, error)
	SearchForMedicationStrength(medicationName string) ([]string, error)
	SelectMedication(medicationName, medicationStrength string) (medication *Medication, err error)
	StartPrescribingPatient(Patient *common.Patient, Treatments []*common.Treatment) error
}

type Medication struct {
	DrugDBIds               map[string]string
	OTC                     bool
	DispenseUnitId          int
	DispenseUnitDescription string
}
