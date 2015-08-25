package doctor_treatment_plan

import (
	"strings"
	"testing"
)

func TestToken_Valid(t *testing.T) {
	tok := newTokenizer('{', '}', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
	})

	// test validation without any tokens
	validateInputWithTokenizer(t, `
		Hello
	`, tok)
	validateInputWithTokenizer(t, ``, tok)

	// test single token validation
	validateInputWithTokenizer(t, `Hello {PTFIRSTNAME}`, tok)
	validateInputWithTokenizer(t, `
		Hello {PTFIRSTNAME},
		SUP
	`, tok)
	validateInputWithTokenizer(t, `
		Hello {PTFIRSTNAME},
		{PTFIRSTNAME}
		{PTFIRSTNAME}
		SUP
	`, tok)

	// test multi-token validation
	tok = newTokenizer('{', '}', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
		{
			tokenType: tokenPatientLastName,
			replacer:  "Joe",
		},
	})
	validateInputWithTokenizer(t, `
		Hello
		{PTLASTNAME}{PTFIRSTNAME}
		SUP
	`, tok)
	validateInputWithTokenizer(t, `
		{PTLASTNAME}{PTFIRSTNAME}
	`, tok)
	validateInputWithTokenizer(t, `
		{PTLASTNAME}{PTFIRSTNAME}
		{PTLASTNAME}{PTFIRSTNAME}
		{PTLASTNAME}{PTFIRSTNAME}
	`, tok)

	// // test token validation where the start
	// // and end delimiters are the same
	tok = newTokenizer('^', '^', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
		{
			tokenType: tokenPatientLastName,
			replacer:  "Joe",
		},
	})
	validateInputWithTokenizer(t, `
		Hello
		^PTLASTNAME^^PTFIRSTNAME^
		SUP
	`, tok)
	validateInputWithTokenizer(t, `
		^PTLASTNAME^^PTFIRSTNAME^
	`, tok)
	validateInputWithTokenizer(t, `
		^PTLASTNAME^^PTFIRSTNAME^
		^PTLASTNAME^^PTFIRSTNAME^
		^PTLASTNAME^^PTFIRSTNAME^
	`, tok)
}

func TestToken_Invalid(t *testing.T) {
	tok := newTokenizer('{', '}', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
	})

	confirmInvalidTokens(t, "{PTFIRSTNAME is an invalid token. Tokens must begin with { and end with }", "Hi {PTFIRSTNAME How r u?", tok)
	confirmInvalidTokens(t, "PTFIRSTNAME} is an invalid token. Tokens must begin with { and end with }", "Hi PTFIRSTNAME} How r u?", tok)
	confirmInvalidTokens(t, "{PTFIRSTNAME is an invalid token. Tokens must begin with { and end with }", "Hi {PTFIRSTNAME", tok)
	confirmInvalidTokens(t, "{PTFIRSTNAME is an invalid token. Tokens must begin with { and end with }", `Hi {PTFIRSTNAME
		hello`, tok)
	confirmInvalidTokens(t, "{PTFIRSTNAME{ is an invalid token. Tokens must begin with { and end with }", `Hi {PTFIRSTNAME{`, tok)
	confirmInvalidTokens(t, "{PTFIRSTNAME is an invalid token. Tokens must begin with { and end with }", `Hi {PTFIRSTNAME {`, tok)
	confirmInvalidTokens(t, "{ is an invalid token. Tokens must begin with { and end with }", `Hi {PTFIRSTNAME} {`, tok)
	confirmInvalidTokens(t, "} is an invalid token. Tokens must begin with { and end with }", `Hi {PTFIRSTNAME}}`, tok)
	confirmInvalidTokens(t, "{PTFIRSTNAME, is an invalid token. Tokens must begin with { and end with }", "Hi {PTFIRSTNAME, {DRLASTNAME}", tok)
	confirmInvalidTokens(t, "DRLASTNAME} is an invalid token. Tokens must begin with { and end with }", "Hi {PTFIRSTNAME},daighagkh DRLASTNAME}", tok)
	confirmInvalidTokens(t, "{TFIRSTNAME} is not a valid token", `{TFIRSTNAME}`, tok)

}

func TestToken_Apply(t *testing.T) {
	tok := newTokenizer('{', '}', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
	})

	// test validation without any tokens
	input := `
		Hello 
	`
	expected := `
		Hello 
	`
	replaceTokensInInput(t, expected, input, tok)

	replaceTokensInInput(t, "", "", tok)

	// test single token validation
	input = `
		Hello {PTFIRSTNAME},
		SUP
	`
	expected = `
		Hello Joe,
		SUP
	`
	replaceTokensInInput(t, expected, input, tok)

	// test single token validation
	input = `Hello {PTFIRSTNAME}`
	expected = `Hello Joe`
	replaceTokensInInput(t, expected, input, tok)

	input = `
		Hello {PTFIRSTNAME},
		{PTFIRSTNAME}
		{PTFIRSTNAME}
		SUP
	`
	expected = `
		Hello Joe,
		Joe
		Joe
		SUP
	`
	replaceTokensInInput(t, expected, input, tok)

	// test multi-token validation
	tok = newTokenizer('{', '}', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
		{
			tokenType: tokenPatientLastName,
			replacer:  "Schmoe",
		},
	})

	input = `
		Hello 
		SchmoeJoe
		SUP
	`
	expected = `
		Hello 
		SchmoeJoe
		SUP
	`
	replaceTokensInInput(t, expected, input, tok)

	input = `
		{PTLASTNAME}{PTFIRSTNAME}
	`
	expected = `
		SchmoeJoe
	`
	replaceTokensInInput(t, expected, input, tok)

	input = `
		{PTLASTNAME}{PTFIRSTNAME}
		{PTLASTNAME}{PTFIRSTNAME}
		{PTLASTNAME}{PTFIRSTNAME}
	`
	expected = `
		SchmoeJoe
		SchmoeJoe
		SchmoeJoe
	`
	replaceTokensInInput(t, expected, input, tok)

	// test token validation where the start
	// and end delimiters are the same
	tok = newTokenizer('^', '^', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
		{
			tokenType: tokenPatientLastName,
			replacer:  "Schmoe",
		},
	})
	input = `
		^PTLASTNAME^^PTFIRSTNAME^
		^PTLASTNAME^^PTFIRSTNAME^
		^PTLASTNAME^^PTFIRSTNAME^
	`
	expected = `
		SchmoeJoe
		SchmoeJoe
		SchmoeJoe
	`
	replaceTokensInInput(t, expected, input, tok)
}

func BenchmarkTokens_Validate(b *testing.B) {
	input := `Hello {PTFIRSTNAME}, 

	I'd like to prescribe you the following medications:
	- Tretinoin
	- Doxycycline
	{PTFIRSTNAME}

	Take each twice a day before breakfast and after dinner. Remember that things get worse before they get better.

	Check in with me in 2 weeks.

	I'd like to prescribe you the following medications:
	- Tretinoin
	- Doxycycline
	{PTFIRSTNAME}

	Take each twice a day before breakfast and after dinner. Remember that things get worse before they get better.

	Check in with me in 2 weeks.

	I'd like to prescribe you the following medications:
	- Tretinoin
	- Doxycycline
	{PTFIRSTNAME}

	Take each twice a day before breakfast and after dinner. Remember that things get worse before they get better.

	Check in with me in 2 weeks.


	I'd like to prescribe you the following medications:
	- Tretinoin
	- Doxycycline

	Take each twice a day before breakfast and after dinner. Remember that things get worse before they get better.

	Check in with me in 2 weeks.

	Adios,
	{DRFULLNAME}`

	t := newTokenizer('{', '}', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
		{
			tokenType: tokenDoctorFullName,
			replacer:  "Dr. Schmoe",
		},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := t.validate(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTokens_Replace(b *testing.B) {
	input := `Hello {PTFIRSTNAME}, 

	I'd like to prescribe you the following medications:
	- Tretinoin
	- Doxycycline
	{PTFIRSTNAME}

	Take each twice a day before breakfast and after dinner. Remember that things get worse before they get better.

	Check in with me in 2 weeks.

	I'd like to prescribe you the following medications:
	- Tretinoin
	- Doxycycline
	{PTFIRSTNAME}

	Take each twice a day before breakfast and after dinner. Remember that things get worse before they get better.

	Check in with me in 2 weeks.

	I'd like to prescribe you the following medications:
	- Tretinoin
	- Doxycycline
	{PTFIRSTNAME}

	Take each twice a day before breakfast and after dinner. Remember that things get worse before they get better.

	Check in with me in 2 weeks.


	I'd like to prescribe you the following medications:
	- Tretinoin
	- Doxycycline

	Take each twice a day before breakfast and after dinner. Remember that things get worse before they get better.

	Check in with me in 2 weeks.

	Adios,
	{DRFULLNAME}`

	t := newTokenizer('{', '}', []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  "Joe",
		},
		{
			tokenType: tokenDoctorFullName,
			replacer:  "Dr. Schmoe",
		},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := t.replace(input); err != nil {
			b.Fatal(err)
		}
	}
}

func validateInputWithTokenizer(t *testing.T, m string, tok *tokenizer) {
	if err := tok.validate(m); err != nil {
		t.Fatal(err)
	}
}

func confirmInvalidTokens(t *testing.T, expectedErrorMsg, m string, tok *tokenizer) {
	if err := tok.validate(m); err == nil {
		t.Fatalf("Expected %s to contain invalid tokens but didnt", m)
	} else if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Fatalf("Expected error message to contain:\n'%s'\nbut instead got\n'%s'", expectedErrorMsg, err.Error())
	}

}

func replaceTokensInInput(t *testing.T, expected, m string, tok *tokenizer) {
	res, err := tok.replace(m)
	if err != nil {
		t.Fatal(err)
	}
	if expected != res {
		t.Fatalf("Expected %s, got %s", expected, res)
	}
}
