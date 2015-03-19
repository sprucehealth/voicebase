package main

import "strings"

func screenParser(p *parser, v string) interface{} {
	scr := &Screen{
		HeaderTitle: v,
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
		case "end screen":
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
		block, eof := p.readBlock([]string{"end screen"}, true)
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
