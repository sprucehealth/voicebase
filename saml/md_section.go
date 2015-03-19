package main

func mdSectionParser(p *parser, v string) interface{} {
	sec := &Subsection{
		Title: v,
	}
	for {
		block, eof := p.readBlock([]string{"md section", "patient section"}, false)
		if eof || block == nil {
			return sec
		}
		switch b := block.(type) {
		default:
			p.err("MD section cannot contain block of type %T", block)
		case comment:
		case *questionBlock:
			sec.Screens = append(sec.Screens, &Screen{
				Questions: []*Question{b.q},
			})
			sec.Screens = append(sec.Screens, b.s...)
		case *Screen:
			sec.Screens = append(sec.Screens, b)
		}
	}
}
