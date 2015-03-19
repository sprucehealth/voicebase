package main

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
		name, value := p.parseSingleDirective(line)
		name = strings.Replace(name, " ", "_", -1)
		switch name {
		default:
			p.err("Unknown view attribute '%s'", name)
		case "end_view":
			return View(view)
		case "element_style", "text":
			view[name] = value
		case "number":
			n, err := strconv.Atoi(value)
			if err != nil {
				p.err("Expected a number for %s, found '%s'", name, value)
			}
			view[name] = n
		}
	}
}
