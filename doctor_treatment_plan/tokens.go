package doctor_treatment_plan

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/errors"
)

type tokenType string

// token is a struct that represents a phrase that if contained
// within the start and end delimiters is intended to be replaced
// from a block of text.
// Note that a token is assumed to not contain white spaces or new lines.
type token struct {
	tokenType
	replacer    string
	description string
}

const (
	tokenPatientFirstName tokenType = "PTFIRSTNAME"
	tokenPatientLastName  tokenType = "PTLASTNAME"
	tokenDoctorFullName   tokenType = "DRFULLNAME"
	tokenDoctorLastName   tokenType = "DRLASTNAME"
)

// tokenizer is a struct that represents a particular structure
// for tokens (determined by start and end delimiter) along with a list of tokens
// to apply to any input string.
type tokenizer struct {
	// startDelimiter represents the rune used to indicate the start of a token.
	// Note that the rune cannot be used in any part of the string other than
	// as the start delimiter.
	startDelimiter rune

	// endDelimiter represens the rune used to indicate the end of a token.
	// Note that the rune cannot be used in any part of the string other than
	// as the end delimiter.
	endDelimiter rune

	// tokens is a mapping of tokenType to the appropriate
	// token. It is used to validate and replace the tokens in the input string
	tokens map[tokenType]*token

	// replacer is used to replace the tokens in the input string.
	replacer *strings.Replacer
}

func newPatientDoctorTokenizer(patient *common.Patient, doctor *common.Doctor) *tokenizer {
	var patientLastNameWithTitle string
	switch strings.ToLower(patient.Gender) {
	case "male":
		patientLastNameWithTitle = "Mr. " + strings.TrimSpace(patient.LastName)
	case "female":
		patientLastNameWithTitle = "Ms. " + strings.TrimSpace(patient.LastName)
	}

	return newTokenizer('{', '}', []*token{
		{
			tokenType:   tokenPatientFirstName,
			description: "Patient's First Name",
			replacer:    strings.TrimSpace(patient.FirstName),
		},
		{
			tokenType:   tokenPatientLastName,
			description: "Patient's Last Name",
			replacer:    patientLastNameWithTitle,
		},
		{
			tokenType:   tokenDoctorLastName,
			description: "Doctor's Last Name",
			replacer:    doctor.ShortDisplayName,
		},
		{
			tokenType:   tokenDoctorFullName,
			description: "Doctor's Full Name",
			replacer:    doctor.LongDisplayName,
		},
	})
}

// newTokenizerForValidation returns a tokenizer initialized
// with default set of tokens to be used for validation purposes
func newTokenizerForValidation(start, end rune) *tokenizer {
	return newTokenizer(start, end, []*token{
		{
			tokenType: tokenPatientFirstName,
			replacer:  string(tokenPatientFirstName),
		},
		{
			tokenType: tokenPatientLastName,
			replacer:  string(tokenPatientLastName),
		},
		{
			tokenType: tokenDoctorLastName,
			replacer:  string(tokenDoctorLastName),
		},
		{
			tokenType: tokenDoctorFullName,
			replacer:  string(tokenDoctorFullName),
		},
	})
}

func newTokenizer(start, end rune, tokens []*token) *tokenizer {
	tokensMap := make(map[tokenType]*token, len(tokens))
	for _, t := range tokens {
		tokensMap[t.tokenType] = t
	}

	return &tokenizer{
		startDelimiter: start,
		endDelimiter:   end,
		tokens:         tokensMap,
	}
}

// validate ensures that there are no invalid tokens in the input
func (t *tokenizer) validate(input string) error {

	p := initParser(input, t)

	// parsing of the input string to valid token works against the following state machine:
	//
	//                  ┌───────────────────▶     ERROR     ◀────────────────────┐
	//                  │                           ▲                            │
	//                  │                           │                            │
	//            no start, but               no end delim                 non-existent
	//           end delim found                  found                        token

	//                  │                           │                            │
	//                  │                           │                            │
	//                  │                           │                            │
	//                  │    start delim found      │       end delim found      │    end of input

	// ───────▶ NEXT START DELIM ─────────▶  NEXT END DELIM  ────────────▶  CHECK TOKEN ──────▶  END

	//                  ▲                                                        │
	//                  │                                                        │
	//                  └────────────────────  token found  ─────────────────────┘

	for state := nextStartDelimiter; state != nil; {
		state = state(p)
	}

	return p.err
}

// stateFn is type that represents a particular step
// of a state machine and returns the next step to execute.
type stateFn func(p *parser) stateFn

// parser is used to hold a reference to point in time during
// parsing of tokenized input.
type parser struct {
	// input represents string to parse and check for valid tokens
	input string

	// idx represents current index in the parsing
	// of the input
	idx int

	// lastStartDelimIdx represents the index of the last start
	// delimiter processed
	lastStartDelimIdx int

	// lastEndDelimIdx represents the index of the last
	// end delimiter processed
	lastEndDelimIdx int

	// t is a reference to the tokenizer
	// that contains the tokens to validate against.
	t *tokenizer

	// err represents any error in the parsing
	// or validating of the input string.
	err error
}

func (p *parser) populateUserError(invalidToken string) {
	msg := fmt.Sprintf("%s is an invalid token. Tokens must begin with %s and end with %s",
		invalidToken, string(p.t.startDelimiter), string(p.t.endDelimiter))

	p.err = userError(msg)
}

func userError(msg string) *apiservice.SpruceError {
	return &apiservice.SpruceError{
		DeveloperError: msg,
		UserError:      msg,
		HTTPStatusCode: http.StatusBadRequest,
	}
}

func initParser(input string, t *tokenizer) *parser {
	return &parser{
		input:             input,
		t:                 t,
		lastStartDelimIdx: -1,
		lastEndDelimIdx:   -1,
	}
}

// nextStartDelimiter looks for the start delimiter from the current position
// in the parser and returns the nextEndDelimiter state if a startDelimiter is
// found and no next state otherwise.
func nextStartDelimiter(p *parser) stateFn {
	input := p.input[p.idx:]
	startIdx := strings.IndexRune(input, p.t.startDelimiter)
	if startIdx == -1 {
		// ensure that end delimiter doesn't exist if no start delimiter is found.
		if endIdx := strings.IndexRune(input, p.t.endDelimiter); endIdx >= 0 {

			// find the last whitespace or new line before the end delimiter to return
			// helpful information to the user on the token that is invalid.
			// for example, if the string is "Hello {JOE}, sincerely SCHMOE}",
			// then an attempt is made to return SCHMOE} as the invalid token.
			var tokenStartIdx int
			if whiteSpaceIdx := lastIndexWhiteSpaceOrNewLine(input[:endIdx]); whiteSpaceIdx >= 0 {
				tokenStartIdx = whiteSpaceIdx + 1
			}

			p.populateUserError(input[tokenStartIdx : endIdx+1])
		}

		// nothing else to find if start delimiter nor end delimiter exists
		return nil
	}

	// update state
	p.lastStartDelimIdx = p.idx + startIdx
	p.idx = p.lastStartDelimIdx + 1

	return nextEndDelimiter
}

// nextEndDelimiter assumes the startDelimiter has been found, and then looks for the end delimiter
// after the start delimiter. If no end delimiter is found, then it is considered an error, otherwise
// checkToken is returned as the next state in the state machine.
func nextEndDelimiter(p *parser) stateFn {
	startIdx := p.lastStartDelimIdx + 1
	input := p.input[startIdx:]

	endIdx := strings.IndexRune(input, p.t.endDelimiter)
	whiteSpaceIdx := indexWhiteSpaceOrNewLine(input)

	// determine which comes first, white space/new line or end delimiter.
	// if whitespace or new line, then token is considered invalid.
	if whiteSpaceIdx >= 0 && endIdx > whiteSpaceIdx {
		p.populateUserError(p.input[p.lastStartDelimIdx : startIdx+whiteSpaceIdx])
		return nil
	} else if endIdx == -1 {
		tokenEndIdx := len(input)
		if whiteSpaceIdx >= 0 {
			tokenEndIdx = whiteSpaceIdx
		}
		p.populateUserError(p.input[p.lastStartDelimIdx : startIdx+tokenEndIdx])
		return nil
	}

	// update state
	p.lastEndDelimIdx = startIdx + endIdx
	p.idx = p.lastEndDelimIdx + 1

	return checkToken
}

// checkToken assumes that the end and start delimiters of a token have been found, and in between
// the positions of the delimiters lies a potential token. The token is looked up to ensure
// that it actually exists as a potential token. If the end of the input string has not been reached,
// nextStartDelimiter is returned as the next step, nil otherwise.
func checkToken(p *parser) stateFn {

	if p.lastStartDelimIdx == -1 || p.lastEndDelimIdx == -1 {
		p.err = errors.New("token boundaries not yet defined")
		return nil
	} else if p.lastStartDelimIdx >= p.lastEndDelimIdx {
		p.err = fmt.Errorf("invalid token boundaries. lastStartDelimIdx: %d lastEndDelimIdx: %d", p.lastStartDelimIdx, p.lastEndDelimIdx)
		return nil
	} else if p.lastEndDelimIdx >= len(p.input) {
		p.err = fmt.Errorf("invalid token boundaries. lastStartDelimIdx: %d length: %d", p.lastStartDelimIdx, len(p.input))
		return nil
	}

	// check if the token actually exists
	tokenString := p.input[p.lastStartDelimIdx+1 : p.lastEndDelimIdx]
	if p.t.tokens[tokenType(tokenString)] == nil {
		p.err = userError(fmt.Sprintf("%s is not a valid token", p.input[p.lastStartDelimIdx:p.lastEndDelimIdx+1]))
		return nil
	}

	// keep looking for tokens unless end of input string
	// has been reached.
	if p.idx < len(p.input) {
		return nextStartDelimiter
	}

	return nil
}

// replace returns a string with all tokens replaced. an error is returned
// if the input contains invalid tokens.
func (t *tokenizer) replace(input string) (string, error) {

	if t.replacer == nil {
		oldnew := make([]string, 0, 2*(len(t.tokens)+2))
		for _, t := range t.tokens {
			oldnew = append(oldnew, string(t.tokenType))
			oldnew = append(oldnew, t.replacer)
		}
		// add the delimiters so that they get replaced with empty strings
		oldnew = append(oldnew, string(t.startDelimiter), "", string(t.endDelimiter), "")
		t.replacer = strings.NewReplacer(oldnew...)
	}

	// first validate to ensure all is good in the world
	if err := t.validate(input); err != nil {
		return "", err
	}

	return t.replacer.Replace(input), nil
}

func indexWhiteSpaceOrNewLine(input string) int {
	whiteSpaceIdx := strings.IndexRune(input, ' ')
	if whiteSpaceIdx >= 0 {
		return whiteSpaceIdx
	}
	return strings.Index(input, "\n")
}

func lastIndexWhiteSpaceOrNewLine(input string) int {
	lastWhiteSpaceIdx := strings.LastIndex(input, " ")
	lastNewLineIdx := strings.LastIndex(input, "\n")

	if lastWhiteSpaceIdx >= 0 && lastNewLineIdx >= 0 {
		if lastWhiteSpaceIdx < lastNewLineIdx {
			return lastWhiteSpaceIdx
		}
	} else if lastWhiteSpaceIdx >= 0 {
		return lastWhiteSpaceIdx
	}
	return lastNewLineIdx
}
