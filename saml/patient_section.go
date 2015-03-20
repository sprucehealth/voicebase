package saml

func patientSectionParser(p *parser, v string) interface{} {
	sec := &Section{
		Title: v,
	}

	// Read attributes
	for {
		line, eof := p.readLine()
		if eof {
			return sec
		}
		if line == "" {
			break
		}
		name, value := p.parseSingleDirective(line)
		switch name {
		default:
			p.err("Unknown patient section directive '%s'", name)
		case "transition message":
			sec.TransitionToMessage = value
		}
	}

	// Read blocks
	for {
		block, eof := p.readBlock([]string{"patient section"}, false)
		if eof || block == nil {
			return sec
		}
		switch b := block.(type) {
		default:
			p.err("Patient section cannot contain block of type %T", block)
		case comment:
		case *Subsection:
			sec.Subsections = append(sec.Subsections, b)
		case *Screen:
			sec.Screens = append(sec.Screens, b)
		case *Question:
			sec.Screens = append(sec.Screens, &Screen{
				Questions: []*Question{b},
			})
		}
	}
}
