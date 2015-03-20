package saml

type triage struct {
	name    string
	screens []*Screen
}

func triageParser(p *parser, v string) interface{} {
	tr := &triage{
		name: v,
	}
	for {
		block, eof := p.readBlock([]string{"end triage"}, true)
		if eof || block == nil {
			if len(tr.screens) == 0 {
				p.err("Triage missing screens")
			}
			return tr
		}
		switch b := block.(type) {
		default:
			p.err("Triage cannot contain block of type %T", block)
		case *Screen:
			tr.screens = append(tr.screens, b)
		}
	}
}
