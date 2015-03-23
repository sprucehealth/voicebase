package saml

import "strings"

type screenTemplate struct {
	name string
	scr  *Screen
}

func screenParser(p *parser, v string) interface{} {
	return screenTypeParser(p, v, "screen", nil)
}

func screenTemplateParser(p *parser, name string) interface{} {
	scr := screenTypeParser(p, "", "screen template", nil).(*Screen)
	return &screenTemplate{name: strings.ToLower(name), scr: scr}
}

func includeScreenParser(p *parser, name string) interface{} {
	scr := p.screenTemplates[strings.ToLower(name)]
	if scr == nil {
		p.err("No screen template named '%s'", name)
	}
	return screenTypeParser(p, "", "include screen", scr.clone())
}

func screenTypeParser(p *parser, v, blockName string, scr *Screen) interface{} {
	if scr == nil {
		scr = &Screen{
			HeaderTitle: v,
		}
	}

	// Read attributes
	for {
		line, eof := p.readLine()
		if eof {
			p.err("Unexpected EOF while parsing screen")
		}
		if line == "" {
			break
		}
		name, value := p.parseSingleDirective(line)
		if strings.HasPrefix(name, "triage ") {
			if scr.ClientData == nil {
				scr.ClientData = &ScreenClientData{}
			}
			if scr.ClientData.Triage == nil {
				scr.ClientData.Triage = &TriageParams{}
			}
		}
		switch name {
		default:
			p.err("Unknown screen directive '%s'", name)
		case "title":
			scr.Title = value
		case "type":
			scr.Type = value
		case "header title":
			scr.HeaderTitle = value
		case "subtitle":
			scr.HeaderSubtitle = value
		case "summary":
			scr.HeaderSummary = value
		case "body text":
			if scr.Body == nil {
				scr.Body = &ScreenBody{}
			}
			scr.Body.Text = value
		case "content header title":
			scr.ContentHeaderTitle = value
		case "bottom button title":
			scr.BottomButtonTitle = value
		case "condition":
			cond, targets := p.parseCondition(value)
			if len(targets) != 0 {
				p.err("A condition directive may not have targets")
			}
			scr.Condition = cond
		case "optional":
			if scr.ClientData == nil {
				scr.ClientData = &ScreenClientData{}
			}
			scr.ClientData.RequiresAtLeastOneQuestionAnswered = boolPtr(false)
		case "end " + blockName:
			return scr
		case "triage abandon":
			scr.ClientData.Triage.Abandon = boolPtr(true)
		case "triage pathway tag":
			if value == "" {
				p.err("Empty triage pathway tag")
			}
			scr.ClientData.PathwayTag = value
		}
	}

	// Read blocks
	for {
		block, eof := p.readBlock([]string{"end " + blockName}, true)
		if eof || block == nil {
			return scr
		}
		switch b := block.(type) {
		default:
			p.err("Screen cannot contain block of type %T", block)
		case comment:
		case *questionBlock:
			scr.Questions = append(scr.Questions, b.q)
			if len(b.s) != 0 {
				// TODO: could support this by returning a slice of screens
				p.err("Screen cannot contain triaged questions")
			}
		case View:
			if scr.ClientData == nil {
				scr.ClientData = &ScreenClientData{}
			}
			scr.ClientData.Views = append(scr.ClientData.Views, b)
		}
	}
}
