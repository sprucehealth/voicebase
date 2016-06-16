package passreset

type promptTemplateContext struct {
	Email        string
	Error        string
	Sent         bool
	SupportEmail string
}

type verifyTemplateContext struct {
	Token         string
	Email         string
	LastTwoDigits string
	EnterCode     bool
	Code          string
	Errors        []string
	SupportEmail  string
}

type resetTemplateContext struct {
	Token        string
	Email        string
	Done         bool
	Errors       []string
	SupportEmail string
}
