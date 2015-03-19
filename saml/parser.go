package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	reDirective  = regexp.MustCompile(`\[[^\]]+\]`)
	reMultiSpace = regexp.MustCompile(`\s{2,}`)
	// Google doc adds annotations/comments/suggestions as "[a]" when exporting to text
	reAnnotation = regexp.MustCompile(`\[[a-z]\]`)
	reCondTag    = regexp.MustCompile(`^[a-zA-Z0-9_\.]+\)`)

	targetDivider      = ","
	targetSeparator    = '→'
	targetSeparatorLen = utf8.RuneLen(targetSeparator)

	// Replace unicode quotes with ASCII quotes
	directiveReplacer = strings.NewReplacer(`“`, `"`, `”`, `"`)
)

type screenContainer interface {
	addScreen(*Screen)
}

type questionContainer interface {
	addQuestion(*Question)
}

type ParseError struct {
	Line int
	Msg  string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("parsing error at line %d: %s", e.Line, e.Msg)
}

type parserState string

const (
	stateNone       parserState = ""
	stateSection    parserState = "section"
	stateSubSection parserState = "sub section"
	stateScreen     parserState = "screen"
	stateQuestion   parserState = "question"
	stateAnswers    parserState = "answers"
	stateComment    parserState = "comment"
	stateTriage     parserState = "triage"
	stateView       parserState = "view"
)

type triage struct {
	popup      string
	nextSteps  string
	help       string
	pathwayTag string
	params     *TriageParams
}

type parseDocCtx struct {
	state      parserState
	intake     *Intake
	sec        *Section
	sub        *Subsection
	scr        *Screen
	que        *Question
	pQue       *Question // previous question for use with the [subquestion] section
	sCont      []screenContainer
	qCont      []questionContainer
	postCond   []string
	triageCond *Condition
	cond       map[string]*Condition // Delayed conditionals for future questions: tag -> conditions
	cTagsUsed  map[string]bool       // Condition tags used (for better error reporting on unused)
	lineNum    int
	qTags      map[string]bool
	triage     map[string]*triage
	triageName string
}

func parseDoc(r io.Reader) (in *Intake, err error) {
	ctx := &parseDocCtx{
		state:     stateNone,
		intake:    &Intake{},
		cond:      make(map[string]*Condition),
		qTags:     make(map[string]bool),
		cTagsUsed: make(map[string]bool),
		triage:    make(map[string]*triage),
	}

	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(ParseError); ok {
				err = er
			} else {
				err = ParseError{Line: ctx.lineNum, Msg: fmt.Sprintf("%+v", e)}
			}
		}
	}()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		ctx.lineNum++

		// Check for BOM
		if ctx.lineNum == 1 {
			if r, n := utf8.DecodeRuneInString(line); r == 0xfeff || r == 0xfffe {
				line = line[n:]
			}
		}

		// Ignore leading and trailing spaces
		line = strings.TrimSpace(line)

		// TODO: ideally these could be excluded from the exported text to begin with
		line = reAnnotation.ReplaceAllString(line, "")

		// Treat page breaks as empty lines
		if line == "________________" {
			line = ""
		}

		// End of algorithm marker. This is optional but useful to ignore the
		// annotations/commntse google docs adds to the end of the text file.
		if line == "[FIN]" {
			break
		}

		switch {
		case line != "" && line[0] == '#':
			// Ignore comment lines
		case ctx.state == stateComment:
			if strings.ToLower(line) == "[end comment]" {
				ctx.state = stateNone
			}
		case ctx.state == stateNone: // command/meta blocks are always "[....]"
			if len(line) == 0 {
				ctx.emptyLine()
			} else if line[0] == '[' {
				ctx.parseTopLevelDirective(line)
			} else if ctx.que == nil {
				ctx.parseStartOfQuestion(line)
			} else {
				ctx.err("Expected start of question or top level directive")
			}
		case ctx.state == stateView:
			ctx.parseView(line)
		case ctx.state == stateSection:
			ctx.parseSection(line)
		case ctx.state == stateScreen:
			ctx.parseScreen(line)
		case ctx.state == stateQuestion:
			if line == "" {
				ctx.emptyLine()
			} else {
				ctx.parseQuestion(line)
			}
		case ctx.state == stateTriage:
			ctx.parseTriage(line)
		default:
			ctx.err("Unexpected state %s", ctx.state)
		}
	}
	ctx.emptyLine()

	// Make sure all condition tags were used
	for tag := range ctx.cTagsUsed {
		delete(ctx.cond, tag)
	}
	if len(ctx.cond) != 0 {
		unused := make([]string, 0, len(ctx.cond))
		for tag := range ctx.cond {
			unused = append(unused, tag)
		}
		if len(unused) != 0 {
			ctx.err("Unused condition tags: %s", strings.Join(unused, ", "))
		}
	}

	return ctx.intake, scanner.Err()
}

func (ctx *parseDocCtx) emptyLine() {
	if len(ctx.postCond) != 0 {
		for _, c := range ctx.postCond {
			cond, targets := ctx.parseCondition(c)
			for _, t := range targets {
				if strings.HasPrefix(t, "triage") {
					if i := strings.IndexByte(t, ':'); i > 0 {
						ctx.triageName = t[i+1:]
					}
					if ctx.triageCond == nil {
						ctx.triageCond = cond
					} else {
						ctx.triageCond = &Condition{
							Op: "or",
							Operands: []*Condition{
								ctx.triageCond,
								cond,
							},
						}
					}
				} else {
					ctx.err("Post condition can only currently be used for triage")
				}
			}
		}
		ctx.postCond = ctx.postCond[:0]
	}

	if ctx.triageCond != nil {
		var triageScreens []*Screen
		if ctx.triageName == "" {
			triageScreens = []*Screen{
				{
					Condition:          ctx.triageCond,
					Type:               "screen_type_warning_popup",
					ContentHeaderTitle: "We're going to have to end your visit here.",
					Body: &ScreenBody{
						Text: "Your symptoms and medical history suggest that you may need more immediate medical attention than we can currently provide. A local emergency department is an appropriate option, as is your primary care provider.",
					},
					BottomButtonTitle: "Next Steps",
				},
				{
					Condition:          ctx.triageCond,
					Type:               "screen_type_triage",
					Title:              "Next Steps",
					ContentHeaderTitle: "You should seek in-person medical evaluation today.",
					Body: &ScreenBody{
						Text: "If you have health insurance, you should contact your insurance company to find out which providers are covered under your plan. Locate your insurance card and call the listed Member Services number. A representative will help you locate your nearest in-network emergency department. If you are too ill to call and do not have someone to assist you, go to the most convenient emergency department.\n\nIf you do not have health insurance, go to the most convenient emergency department.",
					},
					BottomButtonTitle: "I Understand",
				},
			}
		} else {
			t := ctx.triage[ctx.triageName]
			if t == nil {
				ctx.err("No triage defined with name '%s'", ctx.triageName)
			}
			if t.popup == "" || t.nextSteps == "" || t.help == "" {
				ctx.err("Triage '%s' missing one of popup, next steps, or help", ctx.triageName)
			}
			triageScreens = []*Screen{
				{
					Condition:          ctx.triageCond,
					Type:               "screen_type_warning_popup",
					ContentHeaderTitle: "We're going to have to end your visit here.",
					Body: &ScreenBody{
						Text: t.popup,
					},
					BottomButtonTitle: "Next Steps",
				},
				{
					Condition:          ctx.triageCond,
					Type:               "screen_type_triage",
					Title:              "Next Steps",
					ContentHeaderTitle: t.nextSteps,
					Body: &ScreenBody{
						Text: strings.TrimSpace(t.help),
					},
					BottomButtonTitle: "I Understand",
				},
			}
			if t.pathwayTag != "" || t.params != nil {
				s := triageScreens[1]
				s.ClientData = &ScreenClientData{
					PathwayTag: t.pathwayTag,
					Triage:     t.params,
				}
			}
		}
		ctx.addScreens(triageScreens...)
		ctx.triageCond = nil
	}

	// End of block/question
	if ctx.que != nil {
		if err := validateQuestion(ctx.que); err != nil {
			ctx.err("Invalid question: %s", err)
		}
		ctx.pQue = ctx.que
		ctx.que = nil
	}
	ctx.state = stateNone
}

func (ctx *parseDocCtx) parseTopLevelDirective(line string) {
	ctx.trace("parse top level directive")

	if line[len(line)-1] != ']' {
		ctx.err("Missing ] at end of line '%s'", line)
	}
	name, value := ctx.parseSingleDirective(line)

	switch name {
	default:
		ctx.err("Unknown top level directive '%s'", line)
	case "comment":
		ctx.state = stateComment
	case "patient section":
		if len(ctx.sCont) > 1 {
			// TODO: for now this is the only possibility
			ctx.err("Missing [end subquestions]")
		}

		// Patient sections are top level so clear all stacks
		ctx.qCont = nil
		ctx.sCont = nil
		ctx.sub = nil
		ctx.sec = &Section{
			Title: value,
		}
		ctx.intake.Sections = append(ctx.intake.Sections, ctx.sec)
		ctx.sCont = []screenContainer{ctx.sec}
		ctx.state = stateSection
	case "md section":
		if len(ctx.sCont) != 1 {
			// TODO: for now this is the only possibility
			ctx.err("Missing [end subquestions]")
		}
		ctx.sub = &Subsection{
			Title: value,
		}
		if ctx.sec == nil {
			ctx.err("'MD Section' without a preivous 'Patient Section'")
		}
		ctx.sec.Subsections = append(ctx.sec.Subsections, ctx.sub)
		ctx.sCont[len(ctx.sCont)-1] = ctx.sub
	case "subquestions":
		if ctx.pQue == nil {
			ctx.err("Cannot create subquestions with no previous question")
		}
		switch ctx.pQue.Details.Type {
		default:
			ctx.err("Only multiple choice and autocomplete questions may have subquestions")
		case "q_type_multiple_choice", "q_type_autocomplete":
		}
		ctx.pQue.SubquestionConfig = &QuestionSubquestionConfig{}
		ctx.sCont = append(ctx.sCont, ctx.pQue.SubquestionConfig)
		ctx.qCont = append(ctx.qCont, ctx.pQue.SubquestionConfig)
	case "end subquestions":
		if len(ctx.sCont) < 2 {
			ctx.err("[end subequestions] without matching [subquestions]")
		}
		ctx.sCont = ctx.sCont[:len(ctx.sCont)-1]
		ctx.qCont = ctx.qCont[:len(ctx.qCont)-1]
	case "screen":
		ctx.scr = &Screen{
			HeaderTitle: value,
		}
		ctx.sCont[len(ctx.sCont)-1].addScreen(ctx.scr)
		ctx.state = stateScreen
		ctx.qCont = append(ctx.qCont, ctx.scr)
	case "end screen":
		if len(ctx.qCont) == 0 {
			ctx.err("[end screen] without matching [screen]")
		}
		ctx.scr = nil
		ctx.qCont = ctx.qCont[:len(ctx.qCont)-1]
	case "triage":
		ctx.state = stateTriage
		ctx.triageName = value
		if ctx.triage[value] != nil {
			ctx.err("duplicate triage section " + value)
		}
		ctx.triage[value] = &triage{}
	case "view":
		if ctx.scr == nil {
			ctx.err("[view] must be inside of a screen")
		}
		ctx.state = stateView
		if ctx.scr.ClientData == nil {
			ctx.scr.ClientData = &ScreenClientData{}
		}
		view := View(map[string]interface{}{"type": value})
		ctx.scr.ClientData.Views = append(ctx.scr.ClientData.Views, view)
	case "end view":
		if ctx.state != stateView {
			ctx.err("[end view] without matching [view]")
		}
		ctx.state = stateNone
	}
}

func (ctx *parseDocCtx) parseView(line string) {
	ctx.trace("parse view")

	view := ctx.scr.ClientData.Views[len(ctx.scr.ClientData.Views)-1]
	name, value := ctx.parseSingleDirective(line)
	switch name {
	default:
		ctx.err("Unknown view directive '%s'", name)
	case "end view":
		ctx.trace("end view")
		ctx.state = stateNone
	case "element style", "text":
		view[strings.Replace(name, " ", "_", -1)] = value
	case "number":
		i, err := strconv.Atoi(value)
		if err != nil {
			ctx.err("Invalid number")
		}
		view[strings.Replace(name, " ", "_", -1)] = i
	}
}

func (ctx *parseDocCtx) parseSection(line string) {
	ctx.trace("parse section")
	if line == "" {
		ctx.trace("end section")
		ctx.state = stateNone
		return
	}

	name, value := ctx.parseSingleDirective(line)
	switch name {
	default:
		ctx.err("Unknown section directive '%s'", name)
	case "transition message":
		ctx.sec.TransitionToMessage = value
	}
}

func (ctx *parseDocCtx) addQuestions(qs ...*Question) {
	if len(ctx.sCont) == 0 {
		ctx.err("Question encountered before a 'Patient Section'")
	}
	if len(ctx.qCont) == 0 {
		for _, q := range qs {
			s := &Screen{
				Questions: []*Question{q},
			}
			ctx.sCont[len(ctx.sCont)-1].addScreen(s)
		}
	} else {
		for _, q := range qs {
			ctx.qCont[len(ctx.qCont)-1].addQuestion(q)
		}
	}
}

func (ctx *parseDocCtx) addScreens(ss ...*Screen) {
	if len(ctx.sCont) == 0 {
		ctx.err("Screen encountered before a section")
	}
	for _, s := range ss {
		ctx.sCont[len(ctx.sCont)-1].addScreen(s)
	}
}

func (ctx *parseDocCtx) addAnswer(as ...*Answer) {
	if len(ctx.que.Details.AnswerGroups) != 0 {
		ag := ctx.que.Details.AnswerGroups[len(ctx.que.Details.AnswerGroups)-1]
		for _, a := range as {
			ag.Answers = append(ag.Answers, a)
		}
	} else {
		for _, a := range as {
			ctx.que.Details.Answers = append(ctx.que.Details.Answers, a)
		}
	}
}

func (ctx *parseDocCtx) parseStartOfQuestion(line string) {
	ctx.trace("parse start of question")

	ctx.triageName = ""
	ctx.que = &Question{
		Details: &QuestionDetails{},
	}

	// The default question type is single select unless overriding by a directive
	ctx.que.Details.Type = "q_type_single_select"

	// Add the question to the first available of screen, subsection, or screen
	ctx.addQuestions(ctx.que)

	// Parse the level tag: `HPI)`
	var qTag string
	if i := strings.IndexByte(line, ')'); i >= 0 {
		qTag = line[:i]
		line = strings.TrimSpace(line[i+1:])
	}
	if qTag == "" {
		ctx.err("Missing question level tag (e.g. 'HPI)')")
	}

	directives, line := ctx.parseDirectives(line)

	for name := range directives {
		switch name {
		default:
			ctx.err("Unknown question directive [%s]", name)
		case "select many":
			ctx.que.Details.Type = "q_type_multiple_choice"
		case "segmented":
			ctx.que.Details.Type = "q_type_segmented_control"
		case "single entry":
			ctx.que.Details.Type = "q_type_single_entry"
		case "free text":
			ctx.que.Details.Type = "q_type_free_text"
		case "photo":
			ctx.que.Details.Type = "q_type_photo_section"
		case "medication picker":
			ctx.que.Details.ToPrefill = boolPtr(true)
			ctx.que.Details.Type = "q_type_autocomplete"
			ctx.que.Details.AdditionalFields = &QuestionAdditionalFields{
				AddButtonText:    "Add Medication",
				AddText:          "Add Medication",
				EmptyStateText:   "No medications specified",
				PlaceholderText:  "Type to add a medication",
				RemoveButtonText: "Remove Medication",
				SaveButtonText:   "Save",
			}
		case "optional":
			ctx.que.Details.Required = boolPtr(false)
		case "required":
			ctx.que.Details.Required = boolPtr(true)
		}
	}

	ctx.cTagsUsed[qTag] = true
	ctx.que.Condition = ctx.cond[qTag]
	if ctx.que.Condition == nil && strings.IndexByte(qTag, '.') >= 0 && !strings.HasSuffix(qTag, ".info") {
		ctx.err("Condition for tag %s not found", qTag)
	}
	ctx.que.Details.Text = line
	ctx.state = stateQuestion
}

func (ctx *parseDocCtx) parseScreen(line string) {
	ctx.trace("parse screen")
	if line == "" {
		ctx.trace("end parsing screen")
		ctx.state = stateNone
		return
	}

	name, value := ctx.parseSingleDirective(line)
	switch name {
	default:
		ctx.err("Unknown screen directive [%s]", name)
	case "type":
		ctx.scr.Type = value
	case "subtitle":
		ctx.scr.HeaderSubtitle = value
	case "summary":
		ctx.scr.HeaderSummary = value
	case "condition":
		cond, targets := ctx.parseCondition(value)
		if len(targets) != 0 {
			ctx.err("A condition directive may not have targets")
		}
		ctx.scr.Condition = cond
	}
}

func (ctx *parseDocCtx) parseQuestion(line string) {
	ctx.trace("parse question")

	// Question attributes
	if line[0] == '[' {
		ctx.trace("parse question attribute")

		if line[len(line)-1] != ']' {
			ctx.err("Missing ] at end of line '%s'", line)
		}
		name, value := ctx.parseSingleDirective(line)

		if name != "photo slot" && len(ctx.que.Details.PhotoSlots) != 0 {
			ps := ctx.que.Details.PhotoSlots[len(ctx.que.Details.PhotoSlots)-1]
			switch name {
			case "required":
				ps.Required = boolPtr(true)
				return
			case "optional":
				ps.Required = boolPtr(false)
				return
			}
			if ps.ClientData == nil {
				ps.ClientData = &PhotoSlotClientData{}
			}
			switch name {
			default:
				ctx.err("Unknown photo slot attribute '%s'", line)
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
			return
		}

		switch name {
		default:
			ctx.err("Unknown question attribute '%s'", line)
		case "subtitle":
			ctx.que.Details.Subtext = value
		case "summary":
			ctx.que.Details.Summary = value
			if ctx.que.Details.Tag == "" {
				tag := tagFromText(ctx.que.Details.Summary)
				ctx.que.Details.Tag = tag
				// Guarantee a unique tag
				i := 2
				for ctx.qTags[ctx.que.Details.Tag] {
					ctx.que.Details.Tag = fmt.Sprintf("%s_%d", tag, i)
					i++
				}
				ctx.qTags[ctx.que.Details.Tag] = true
			}
		case "help":
			if ctx.que.Details.AdditionalFields == nil {
				ctx.que.Details.AdditionalFields = &QuestionAdditionalFields{}
			}
			if ctx.que.Details.AdditionalFields.Popup == nil {
				ctx.que.Details.AdditionalFields.Popup = &Popup{}
			}
			ctx.que.Details.AdditionalFields.Popup.Text = value
		case "placeholder":
			if ctx.que.Details.AdditionalFields == nil {
				ctx.que.Details.AdditionalFields = &QuestionAdditionalFields{}
			}
			ctx.que.Details.AdditionalFields.PlaceholderText = value
		case "condition":
			cond, targets := ctx.parseCondition(value)
			if len(targets) != 0 {
				ctx.err("A condition directive may not have targets")
			}
			if ctx.que.Condition == nil {
				ctx.que.Condition = cond
			} else {
				ctx.que.Condition = &Condition{
					Op: "and",
					Operands: []*Condition{
						ctx.que.Condition,
						cond,
					},
				}
			}
		case "post condition":
			ctx.postCond = append(ctx.postCond, value)
		case "answer group":
			ctx.que.Details.AnswerGroups = append(ctx.que.Details.AnswerGroups, &AnswerGroup{Title: value})
		case "photo slot":
			ctx.que.Details.PhotoSlots = append(ctx.que.Details.PhotoSlots, &PhotoSlot{
				Name: value,
			})
		case "alert":
			ctx.que.Details.AlertText = value
			ctx.que.Details.ToAlert = boolPtr(true)
		case "allows multiple sections":
			if ctx.que.Details.AdditionalFields == nil {
				ctx.que.Details.AdditionalFields = &QuestionAdditionalFields{}
			}
			ctx.que.Details.AdditionalFields.AllowsMultipleSections = boolPtr(true)
		case "user defined section title":
			if ctx.que.Details.AdditionalFields == nil {
				ctx.que.Details.AdditionalFields = &QuestionAdditionalFields{}
			}
			ctx.que.Details.AdditionalFields.UserDefinedSectionTitle = boolPtr(true)
		case "global":
			ctx.que.Details.Global = boolPtr(true)
		case "tag":
			ctx.que.Details.Tag = value
		case "empty state":
			if ctx.que.Details.AdditionalFields == nil {
				ctx.que.Details.AdditionalFields = &QuestionAdditionalFields{}
			}
			ctx.que.Details.AdditionalFields.EmptyStateText = value
		}
		return
	}

	if ctx.que.Details.Summary == "" {
		ctx.err("Question missing summary")
	}

	// Answer
	ans := &Answer{}
	ctx.addAnswer(ans)

	// Make sure we didn't run into another question starting with "Cond)"
	if reCondTag.MatchString(line) {
		ctx.err("Questions run together. Should be new line before '%s'", line)
	}

	// Answer directives
	directives, line := ctx.parseDirectives(line)
	for name, value := range directives {
		switch name {
		default:
			ctx.err("Unknown answer directive [%s]", name)
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

	// Check for a conditional
	if ix := strings.IndexRune(line, targetSeparator); ix >= 0 {
		targets := strings.Split(strings.TrimSpace(line[ix+targetSeparatorLen:]), targetDivider)
		line = strings.TrimSpace(line[:ix])

		if ans.Tag == "" {
			ans.Tag = ctx.que.Details.Tag + "_" + tagFromText(line)
		}

		for _, t := range targets {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}

			if strings.HasPrefix(t, "triage") {
				if i := strings.IndexByte(t, ':'); i > 0 {
					ctx.triageName = t[i+1:]
				}
				if ctx.triageCond != nil {
					if ctx.triageCond.Op == "answer_contains_any" {
						ctx.triageCond.PotentialAnswers = append(ctx.triageCond.PotentialAnswers, ans.Tag)
					} else {
						ctx.triageCond = &Condition{
							Op: "or",
							Operands: []*Condition{
								ctx.triageCond,
								&Condition{
									Op:               "answer_contains_any",
									Question:         ctx.que.Details.Tag,
									PotentialAnswers: []string{ans.Tag},
								},
							},
						}
					}
				} else {
					ctx.triageCond = &Condition{
						Op:               "answer_contains_any",
						Question:         ctx.que.Details.Tag,
						PotentialAnswers: []string{ans.Tag},
					}
				}
			} else {
				cond := ctx.cond[t]
				if cond != nil {
					if cond.Op == "answer_contains_any" && cond.Question == ctx.que.Details.Tag {
						cond.PotentialAnswers = append(cond.PotentialAnswers, ans.Tag)
					} else if cond.Op == "or" && cond.Operands[1].Op == "answer_contains_any" && cond.Operands[1].Question == ctx.que.Details.Tag {
						// Optimize by merging OR cases
						cond.Operands[1].PotentialAnswers = append(cond.Operands[1].PotentialAnswers, ans.Tag)
					} else {
						cond = &Condition{
							Op: "or",
							Operands: []*Condition{
								cond,
								&Condition{
									Op:               "answer_contains_any",
									Question:         ctx.que.Details.Tag,
									PotentialAnswers: []string{ans.Tag},
								},
							},
						}
					}
				} else {
					cond = &Condition{
						Op:               "answer_contains_any",
						Question:         ctx.que.Details.Tag,
						PotentialAnswers: []string{ans.Tag},
					}
				}
				ctx.cond[t] = cond
			}
		}
	}
	if ans.Tag == "" {
		ans.Tag = ctx.que.Details.Tag + "_" + tagFromText(line)
	}
	ans.Text = line
}

func (ctx *parseDocCtx) parseTriage(line string) {
	t := ctx.triage[ctx.triageName]
	if len(line) != 0 && line[0] == '[' {
		name, value := ctx.parseSingleDirective(line)
		switch name {
		default:
			ctx.err("Unknown triage directive [%s]", name)
		case "pop-up":
			t.popup = value
		case "next steps":
			t.nextSteps = value
		case "end triage":
			ctx.state = stateNone
		case "pathway tag":
			t.pathwayTag = value
		case "abandon":
			if t.params == nil {
				t.params = &TriageParams{}
			}
			t.params.Abandon = boolPtr(true)
		case "action message":
			if t.params == nil {
				t.params = &TriageParams{}
			}
			t.params.ActionMessage = value
		case "action url":
			if t.params == nil {
				t.params = &TriageParams{}
			}
			t.params.ActionURL = value
		}
		return
	}
	if t.help != "" {
		t.help += "\n"
	}
	t.help += line
}

// parseDirectives parses out any [directives] from the line and returns
// the line with them removed
func (ctx *parseDocCtx) parseDirectives(line string) (map[string]string, string) {
	directives := make(map[string]string)
	line = reDirective.ReplaceAllStringFunc(line, func(dir string) string {
		dir = dir[1 : len(dir)-1]
		dir = directiveReplacer.Replace(dir)

		if ix := strings.IndexRune(dir, '"'); ix > 0 {
			name := strings.ToLower(strings.TrimSpace(dir[:ix]))
			value, err := strconv.Unquote(strings.TrimSpace(dir[ix:]))
			if err != nil {
				ctx.err("Bad directive %s: %s", dir, err.Error())
			}
			directives[name] = value
		} else {
			directives[strings.ToLower(dir)] = ""
		}
		return " "
	})
	// Replace any run of spaces longer than 1 with a single space
	line = strings.TrimSpace(reMultiSpace.ReplaceAllString(line, " "))
	return directives, line
}

func (ctx *parseDocCtx) parseSingleDirective(line string) (string, string) {
	directives, _ := ctx.parseDirectives(line)
	if len(directives) != 1 {
		ctx.err("Expected a single directive")
	}
	for n, v := range directives {
		return n, v
	}
	panic("shouldn't get here")
}

func (ctx *parseDocCtx) parseCondition(s string) (*Condition, []string) {
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
	cond := ctx.parseCondTokens(tokens)
	if cond == nil {
		ctx.err("Empty condition")
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

func (ctx *parseDocCtx) parseCondTokens(tokens []string) *Condition {
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
				ctx.err("Missing op")
			}
			rightCond := ctx.parseCondTokens(tokens[ix+1:])
			if rightCond == nil {
				ctx.err("Missing term after 'not'")
			}
			return &Condition{
				Op:       "not",
				Operands: []*Condition{rightCond},
			}
		case "and", "or":
			if leftCond == nil {
				ctx.err("Missing term before '%s'", tok)
			}
			rightCond := ctx.parseCondTokens(tokens[ix+1:])
			if rightCond == nil {
				ctx.err("Missing term after '%s'", tok)
			}
			return &Condition{
				Op:       tok,
				Operands: []*Condition{leftCond, rightCond},
			}
		case "male", "female":
			if leftCond != nil {
				ctx.err("Missing op")
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
				ctx.err("Left paren missing matching right paren")
			}
			c := ctx.parseCondTokens(tokens[ix+1 : closingIndex])
			if leftCond != nil {
				return c
			}
			leftCond = c
			ix = closingIndex
		default:
			if leftCond != nil {
				ctx.err("Missing op")
			}

			// Token should in this case be a tag
			ctx.cTagsUsed[tok] = true
			leftCond = ctx.cond[tok]
			if leftCond == nil {
				ctx.err("Unknown condition tag '%s'", tok)
			}
		}
		ix++
	}

	return leftCond
}

func (ctx *parseDocCtx) err(fm string, args ...interface{}) {
	panic(ParseError{
		Line: ctx.lineNum,
		Msg:  fmt.Sprintf(fm, args...),
	})
}

func (ctx *parseDocCtx) trace(fm string, args ...interface{}) {
	if *flagDebug {
		fmt.Printf("TRACE %d: %s\n", ctx.lineNum, fmt.Sprintf(fm, args...))
	}
}

var (
	reTagRemove = regexp.MustCompile(`[^\w\s-]`)
	reTagSpaces = regexp.MustCompile(`[-\s]+`)
)

func tagFromText(v string) string {
	v = reTagRemove.ReplaceAllString(v, "")
	v = strings.ToLower(v)
	v = reTagSpaces.ReplaceAllString(v, "_")
	return v
}

func boolPtr(b bool) *bool {
	return &b
}

func validateQuestion(q *Question) error {
	switch q.Details.Type {
	case "q_type_single_select", "q_type_multiple_choice":
		if q.Details.Summary == "" {
			return errors.New("missing summary text")
		}
		if len(q.Details.Answers) == 0 && len(q.Details.AnswerGroups) == 0 {
			return errors.New("missing potential answers")
		}
	case "q_type_free_text":
		if len(q.Details.Answers) != 0 || len(q.Details.AnswerGroups) != 0 {
			return errors.New("free text questions cannot have potential answers")
		}
	}
	return nil
}
