package saml

import "strings"

type comment []string

func commentParser(p *parser, v string) interface{} {
	var lines []string
	for {
		line, eof := p.readLine()
		if eof {
			p.err("unexpected EOF while parsing comment block")
		}
		if strings.ToLower(line) == "[end comment]" {
			return comment(lines)
		}
		lines = append(lines, line)
	}
}
