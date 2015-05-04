package response

type AssociationDataType string

var (
	CaseAssociationType = AssociationDataType("case")
)

type AssociationDescription interface{}

type TagAssociation struct {
	ID          int64                  `json:"id,string"`
	Description AssociationDescription `json:"description"`
	Type        AssociationDataType    `json:"type"`
}

type PHISafeCaseAssociationDescription struct {
	PatientInitials string `json:"patient_initials"`
	Pathway         string `json:"pathway"`
}

type Tag struct {
	ID   int64  `json:"id,string"`
	Text string `json:"text"`
}
