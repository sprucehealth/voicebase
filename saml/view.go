package saml

import (
	"strconv"
	"strings"
)

func viewParser(p *parser, v string) interface{} {
	view := map[string]interface{}{"type": v}
	for {
		line, eof := p.readLine()
		if eof {
			p.err("Unexpected EOF while parsing screen")
		}
		if line == "" {
			continue
		}
		dir := p.parseSingleDirective(line)
		dir.name = strings.Replace(dir.name, " ", "_", -1)
		switch dir.name {
		default:
			p.err("Unknown view attribute '%s'", dir.name)
		case "end_view":
			return View(view)
		case "element_style", "text":
			view[dir.name] = dir.value
		case "number":
			n, err := strconv.Atoi(dir.value)
			if err != nil {
				p.err("Expected a number for %s, found '%s'", dir.name, dir.value)
			}
			view[dir.name] = n
		}
	}
}
