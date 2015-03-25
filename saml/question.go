package saml

import (
	"fmt"
	"strings"
)

type questionBlock struct {
	q *Question
	s []*Screen // Synthesized screens (triage)
}

func questionParser(p *parser, line string) interface{} {
	triage := make(map[string]*Condition)
	var postCond []string
	que := &Question{
		Details: &QuestionDetails{
			// The default question type is single select unless overriding by a directive
			Type: "q_type_single_select",
		},
	}

	// Parse the level tag: `HPI)`
	var qTag string
	if i := strings.IndexByte(line, ')'); i >= 0 {
		qTag = line[:i]
		line = strings.TrimSpace(line[i+1:])
	}
	if qTag == "" {
		p.err("Missing question level tag (e.g. 'HPI)')")
	}

	directives, line := p.parseDirectives(line)
	for name := range directives {
		switch name {
		default:
			p.err("Unknown question directive [%s]", name)
		case "select many":
			que.Details.Type = "q_type_multiple_choice"
		case "segmented":
			que.Details.Type = "q_type_segmented_control"
		case "single entry":
			que.Details.Type = "q_type_single_entry"
		case "free text":
			que.Details.Type = "q_type_free_text"
		case "photo":
			que.Details.Type = "q_type_photo_section"
		case "medication picker":
			que.Details.ToPrefill = boolPtr(true)
			que.Details.Type = "q_type_autocomplete"
			que.Details.AdditionalFields = &QuestionAdditionalFields{
				AddButtonText:    "Add Medication",
				AddText:          "Add Medication",
				EmptyStateText:   "No medications specified",
				PlaceholderText:  "Type to add a medication",
				RemoveButtonText: "Remove Medication",
				SaveButtonText:   "Save",
			}
		case "optional":
			que.Details.Required = boolPtr(false)
		case "required":
			que.Details.Required = boolPtr(true)
		}
	}

	p.cTagsUsed[qTag] = true
	que.Condition = p.cond[qTag]
	if que.Condition == nil && strings.IndexByte(qTag, '.') >= 0 && !strings.HasSuffix(qTag, ".info") {
		p.err("Condition for tag %s not found", qTag)
	}
	que.Details.Text = line

	for {
		line, eof := p.readLine()
		if eof || line == "" {
			break
		}

		// Question attributes
		if line[0] == '[' {
			if line[len(line)-1] != ']' {
				p.err("Missing ] at end of line '%s'", line)
			}

			name, value := p.parseSingleDirective(line)
			if name != "photo slot" && len(que.Details.PhotoSlots) != 0 {
				ps := que.Details.PhotoSlots[len(que.Details.PhotoSlots)-1]
				switch name {
				case "required":
					ps.Required = boolPtr(true)
					continue
				case "optional":
					ps.Required = boolPtr(false)
					continue
				}
				if ps.ClientData == nil {
					ps.ClientData = &PhotoSlotClientData{}
				}
				switch name {
				default:
					p.err("Unknown photo slot attribute '%s'", line)
				case "tip":
					ps.ClientData.Tip = value
				case "tip subtext":
					ps.ClientData.TipSubtext = value
				case "tip style":
					ps.ClientData.TipStyle = value
				case "overlay image url":
					ps.ClientData.OverlayImageURL = value
				case "photo missing error message":
					ps.ClientData.PhotoMissingErrorMessage = value
				case "initial camera direction":
					ps.ClientData.InitialCameraDirection = value
				case "flash on":
					ps.ClientData.Flash = boolPtr(true)
				}
				continue
			}

			switch name {
			default:
				p.err("Unknown question attribute '%s'", line)
			case "subtitle":
				que.Details.Subtext = value
			case "summary":
				que.Details.Summary = value
				if que.Details.Tag == "" {
					tag := tagFromText(que.Details.Summary)
					que.Details.Tag = tag
					// Guarantee a unique tag
					i := 2
					for p.qTags[que.Details.Tag] {
						que.Details.Tag = fmt.Sprintf("%s_%d", tag, i)
						i++
					}
					p.qTags[que.Details.Tag] = true
				}
			case "help":
				if que.Details.AdditionalFields == nil {
					que.Details.AdditionalFields = &QuestionAdditionalFields{}
				}
				if que.Details.AdditionalFields.Popup == nil {
					que.Details.AdditionalFields.Popup = &Popup{}
				}
				que.Details.AdditionalFields.Popup.Text = value
			case "placeholder":
				if que.Details.AdditionalFields == nil {
					que.Details.AdditionalFields = &QuestionAdditionalFields{}
				}
				que.Details.AdditionalFields.PlaceholderText = value
			case "condition":
				cond, targets := p.parseCondition(value)
				if len(targets) != 0 {
					p.err("A condition directive may not have targets")
				}
				if que.Condition == nil {
					que.Condition = cond
				} else {
					que.Condition = &Condition{
						Op: "and",
						Operands: []*Condition{
							que.Condition,
							cond,
						},
					}
				}
			case "post condition":
				postCond = append(postCond, value)
			case "answer group":
				que.Details.AnswerGroups = append(que.Details.AnswerGroups, &AnswerGroup{Title: value})
			case "photo slot":
				que.Details.PhotoSlots = append(que.Details.PhotoSlots, &PhotoSlot{
					Name: value,
				})
			case "alert":
				que.Details.AlertText = value
				que.Details.ToAlert = boolPtr(true)
			case "allows multiple sections":
				if que.Details.AdditionalFields == nil {
					que.Details.AdditionalFields = &QuestionAdditionalFields{}
				}
				que.Details.AdditionalFields.AllowsMultipleSections = boolPtr(true)
			case "user defined section title":
				if que.Details.AdditionalFields == nil {
					que.Details.AdditionalFields = &QuestionAdditionalFields{}
				}
				que.Details.AdditionalFields.UserDefinedSectionTitle = boolPtr(true)
			case "prefill":
				que.Details.ToPrefill = boolPtr(true)
			case "global":
				que.Details.Global = boolPtr(true)
			case "tag":
				que.Details.Tag = value
			case "empty state":
				if que.Details.AdditionalFields == nil {
					que.Details.AdditionalFields = &QuestionAdditionalFields{}
				}
				que.Details.AdditionalFields.EmptyStateText = value
			}
			continue
		}

		if que.Details.Summary == "" {
			p.err("Question missing summary")
		}

		// Answer
		ans := &Answer{}
		if len(que.Details.AnswerGroups) != 0 {
			ag := que.Details.AnswerGroups[len(que.Details.AnswerGroups)-1]
			ag.Answers = append(ag.Answers, ans)
		} else {
			que.Details.Answers = append(que.Details.Answers, ans)
		}

		// Make sure we didn't run into another question starting with "Cond)"
		if reCondTag.MatchString(line) {
			p.err("Questions run together. Should be new line before '%s'", line)
		}

		// Answer directives
		directives, line := p.parseDirectives(line)
		for name, value := range directives {
			switch name {
			default:
				p.err("Unknown answer directive [%s]", name)
			case "placeholder":
				if ans.ClientData == nil {
					ans.ClientData = &AnswerClientData{}
				}
				ans.ClientData.PlaceholderText = value
			case "help":
				if ans.ClientData == nil {
					ans.ClientData = &AnswerClientData{}
				}
				if ans.ClientData.Popup == nil {
					ans.ClientData.Popup = &Popup{}
				}
				ans.ClientData.Popup.Text = value
			case "summary":
				ans.Summary = value
			case "tag":
				ans.Tag = value
			case "textbox":
				ans.Type = "a_type_multiple_choice_other_free_text"
			case "none":
				ans.Type = "a_type_multiple_choice_none"
			case "alert":
				ans.ToAlert = boolPtr(true)
			}
		}

		if strings.IndexByte(line, '[') != -1 || strings.IndexByte(line, ']') != -1 {
			p.err("Broken directive (missing opening or closing bracket)")
		}

		// Check for a conditional
		if ix := strings.IndexRune(line, targetSeparator); ix >= 0 {
			targets := strings.Split(strings.TrimSpace(line[ix+targetSeparatorLen:]), targetDivider)
			line = strings.TrimSpace(line[:ix])

			if ans.Tag == "" {
				ans.Tag = que.Details.Tag + "_" + tagFromText(line)
			}

			for _, t := range targets {
				t = strings.TrimSpace(t)
				if t == "" {
					continue
				}

				if strings.HasPrefix(t, "triage") {
					triageName := t
					if i := strings.IndexByte(t, ':'); i > 0 {
						triageName = t[i+1:]
					}
					triageCond := triage[triageName]
					if triageCond != nil {
						if triageCond.Op == "answer_contains_any" {
							triageCond.PotentialAnswers = append(triageCond.PotentialAnswers, ans.Tag)
						} else {
							triageCond = &Condition{
								Op: "or",
								Operands: []*Condition{
									triageCond,
									&Condition{
										Op:               "answer_contains_any",
										Question:         que.Details.Tag,
										PotentialAnswers: []string{ans.Tag},
									},
								},
							}
						}
					} else {
						triageCond = &Condition{
							Op:               "answer_contains_any",
							Question:         que.Details.Tag,
							PotentialAnswers: []string{ans.Tag},
						}
					}
					triage[triageName] = triageCond
				} else {
					cond := p.cond[t]
					if cond != nil {
						if cond.Op == "answer_contains_any" && cond.Question == que.Details.Tag {
							cond.PotentialAnswers = append(cond.PotentialAnswers, ans.Tag)
						} else if cond.Op == "or" && cond.Operands[1].Op == "answer_contains_any" && cond.Operands[1].Question == que.Details.Tag {
							// Optimize by merging OR cases
							cond.Operands[1].PotentialAnswers = append(cond.Operands[1].PotentialAnswers, ans.Tag)
						} else {
							cond = &Condition{
								Op: "or",
								Operands: []*Condition{
									cond,
									&Condition{
										Op:               "answer_contains_any",
										Question:         que.Details.Tag,
										PotentialAnswers: []string{ans.Tag},
									},
								},
							}
						}
					} else {
						cond = &Condition{
							Op:               "answer_contains_any",
							Question:         que.Details.Tag,
							PotentialAnswers: []string{ans.Tag},
						}
					}
					p.cond[t] = cond
				}
			}
		}
		if ans.Tag == "" {
			ans.Tag = que.Details.Tag + "_" + tagFromText(line)
		}
		if line == "" {
			p.err("Answer missing text")
		}
		ans.Text = line
	}

	qb := &questionBlock{
		q: que,
	}

	if len(postCond) != 0 {
		for _, c := range postCond {
			cond, targets := p.parseCondition(c)
			for _, t := range targets {
				if strings.HasPrefix(t, "triage") {
					triageName := t
					if i := strings.IndexByte(t, ':'); i > 0 {
						triageName = t[i+1:]
					}
					triageCond := triage[triageName]
					if triageCond == nil {
						triageCond = cond
					} else {
						triageCond = &Condition{
							Op: "or",
							Operands: []*Condition{
								triageCond,
								cond,
							},
						}
					}
					triage[triageName] = triageCond
				} else {
					p.err("Post condition can only currently be used for triage")
				}
			}
		}
	}

	for triageName, triageCond := range triage {
		// TODO: remove the default 'triage out' version once all pathway docs have been
		// updated to use named triage steps
		if triageName == "triage out" {
			qb.s = append(qb.s, &Screen{
				Condition:          triageCond,
				Type:               "screen_type_warning_popup",
				ContentHeaderTitle: "We're going to have to end your visit here.",
				Body: &ScreenBody{
					Text: "Your symptoms and medical history suggest that you may need more immediate medical attention than we can currently provide. A local emergency department is an appropriate option, as is your primary care provider.",
				},
				BottomButtonTitle: "Next Steps",
			},
				&Screen{
					Condition:          triageCond,
					Type:               "screen_type_triage",
					Title:              "Next Steps",
					ContentHeaderTitle: "You should seek in-person medical evaluation today.",
					Body: &ScreenBody{
						Text: "If you have health insurance, you should contact your insurance company to find out which providers are covered under your plan. Locate your insurance card and call the listed Member Services number. A representative will help you locate your nearest in-network emergency department. If you are too ill to call and do not have someone to assist you, go to the most convenient emergency department.\n\nIf you do not have health insurance, go to the most convenient emergency department.",
					},
					BottomButtonTitle: "I Understand",
				},
			)
		} else {
			t := p.triage[triageName]
			if t == nil {
				p.err("No triage defined with name '%s'", triageName)
			}
			for _, s := range t.screens {
				scr := s.clone()
				scr.Condition = triageCond
				qb.s = append(qb.s, scr)
			}
		}
	}

	if err := validateQuestion(que); err != nil {
		p.err("Invalid question: %s", err)
	}

	if p.checkForBlock("subquestions") {
		b, _ := p.readBlock(nil, false)
		que.SubquestionConfig = b.(*QuestionSubquestionConfig)
	}

	return qb
}
