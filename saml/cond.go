package main

import "strings"

func (p *parser) parseCondition(s string) (*Condition, []string) {
	var targets []string
	ix := strings.IndexRune(s, targetSeparator)
	if ix > 0 {
		for _, t := range strings.Split(s[ix+targetSeparatorLen:], targetDivider) {
			t = strings.TrimSpace(t)
			if t != "" {
				targets = append(targets, t)
			}
		}
		s = s[:ix]
	}
	s = strings.TrimSpace(s)

	tokens := tokenizeCondition(s)
	cond := p.parseCondTokens(tokens)
	if cond == nil {
		p.err("Empty condition")
	}
	return cond, targets
}

func tokenizeCondition(s string) []string {
	var tokens []string
	ix := 0
	sx := -1
	for _, r := range s {
		switch r {
		case '(', ')', ' ':
			if sx >= 0 {
				tokens = append(tokens, s[sx:ix])
				sx = -1
			}
			if r != ' ' {
				tokens = append(tokens, string(r))
			}
		default:
			if sx < 0 {
				sx = ix
			}
		}
		ix++
	}
	if sx >= 0 {
		tokens = append(tokens, s[sx:ix])
	}
	return tokens
}

func (p *parser) parseCondTokens(tokens []string) *Condition {
	if len(tokens) == 0 {
		return nil
	}

	var leftCond *Condition
	ix := 0
	for ix < len(tokens) {
		tok := tokens[ix]
		switch tok {
		case "not":
			if leftCond != nil {
				p.err("Missing op")
			}
			rightCond := p.parseCondTokens(tokens[ix+1:])
			if rightCond == nil {
				p.err("Missing term after 'not'")
			}
			return &Condition{
				Op:       "not",
				Operands: []*Condition{rightCond},
			}
		case "and", "or":
			if leftCond == nil {
				p.err("Missing term before '%s'", tok)
			}
			rightCond := p.parseCondTokens(tokens[ix+1:])
			if rightCond == nil {
				p.err("Missing term after '%s'", tok)
			}
			return &Condition{
				Op:       tok,
				Operands: []*Condition{leftCond, rightCond},
			}
		case "male", "female":
			if leftCond != nil {
				p.err("Missing op")
			}
			leftCond = &Condition{
				Op:     "gender_equals",
				Gender: tok,
			}
		case "(":
			var closingIndex int
			depth := 1
			for j, t := range tokens[ix+1:] {
				if t == "(" {
					depth++
				} else if t == ")" {
					depth--
					if depth == 0 {
						closingIndex = ix + j + 1
						break
					}
				}
			}
			if closingIndex == 0 {
				p.err("Left paren missing matching right paren")
			}
			c := p.parseCondTokens(tokens[ix+1 : closingIndex])
			if leftCond != nil {
				return c
			}
			leftCond = c
			ix = closingIndex
		default:
			if leftCond != nil {
				p.err("Missing op")
			}

			// Token should in this case be a tag
			p.cTagsUsed[tok] = true
			leftCond = p.cond[tok]
			if leftCond == nil {
				p.err("Unknown condition tag '%s'", tok)
			}
		}
		ix++
	}

	return leftCond
}
