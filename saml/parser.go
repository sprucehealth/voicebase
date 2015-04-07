package saml

import (
	"bufio"
	"fmt"
	"io"
	"os"
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

var Debug = false

type ParseError struct {
	Line int
	Msg  string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("parsing error at line %d: %s", e.Line, e.Msg)
}

type blockParser func(*parser, string) interface{}

var blockParsers map[string]blockParser

func init() {
	blockParsers = map[string]blockParser{
		"comment":         commentParser,
		"triage":          triageParser,
		"patient section": patientSectionParser,
		"md section":      mdSectionParser,
		"screen":          screenParser,
		"screen template": screenTemplateParser,
		"include screen":  includeScreenParser,
		"view":            viewParser,
		"subquestions":    subquestionsParser,
	}
}

type parser struct {
	lineNum         int
	scanner         *bufio.Scanner
	fin             bool
	storedLine      string
	intake          *Intake
	cond            map[string]*Condition // Delayed conditionals for future questions: condition tag -> conditions
	cTagsUsed       map[string]bool       // Condition tags used (for reporting unused tags)
	qTags           map[string]bool       // Seen question tags. Used to guarantee uniqueness of generated tags.
	triage          map[string]*triage
	screenTemplates map[string]*Screen
}

func Parse(r io.Reader) (in *Intake, err error) {
	parser := &parser{
		intake:          &Intake{},
		cond:            make(map[string]*Condition),
		qTags:           make(map[string]bool),
		cTagsUsed:       make(map[string]bool),
		triage:          make(map[string]*triage),
		screenTemplates: make(map[string]*Screen),
		scanner:         bufio.NewScanner(r),
	}

	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(ParseError); ok {
				err = er
			} else {
				err = ParseError{Line: parser.lineNum, Msg: fmt.Sprintf("%+v", e)}
			}
		}
	}()

	for {
		block, eof := parser.readBlock(nil, false)
		if eof {
			break
		}
		switch b := block.(type) {
		default:
			parser.err("Unsupported top-level block type %T", block)
		case comment:
		case *triage:
			parser.triage[b.name] = b
		case *Section:
			parser.intake.Sections = append(parser.intake.Sections, b)
		case *screenTemplate:
			parser.screenTemplates[b.name] = b.scr
		}
	}

	return parser.intake, nil
}

// checkForBlock scans ahead and looks for a top level directive with the
// given name. It reports whether the block was found or not. It doesn't
// consume any input.
func (p *parser) checkForBlock(name string) bool {
	for {
		line, eof := p.readLine()
		if eof {
			return false
		}
		if len(line) == 0 {
			continue
		}
		p.storedLine = line

		if line[0] == '[' {
			if line[len(line)-1] != ']' {
				p.err("Missing ] at end of line '%s'", line)
			}
			n, _ := p.parseSingleDirective(line)
			if n == name {
				return true
			}
		}
		return false
	}
}

// readBlock reads the next block from the input and returns a bool
// that indicates whether the end of the input has been reached. If
// endMarkers is given then if before a block is encountered one of
// the markers is found (matching the directive name) then it returns
// nil for the block and false for EOF. If consumeEnd is true then
// if an end marker is hit the line is consumed, otherwise the
// marker line that was hit is not consumed and will be used for the
// next read.
func (p *parser) readBlock(endMarkers []string, consumeEnd bool) (interface{}, bool) {
	for {
		line, eof := p.readLine()
		if eof {
			return nil, true
		}

		if len(line) == 0 {
			continue
		}

		if line[0] == '[' {
			if line[len(line)-1] != ']' {
				p.err("Missing ] at end of line '%s'", line)

			}

			name, value := p.parseSingleDirective(line)
			for _, m := range endMarkers {
				if m == name {
					if !consumeEnd {
						p.storedLine = line
					}
					return nil, false
				}
			}

			bp := blockParsers[name]
			if bp == nil {
				p.err("Unknown top level directive '%s'", line)
			}
			p.trace("PARSING " + name)
			return bp(p, value), false
		}

		p.trace("PARSING question")
		return questionParser(p, line), false
	}
}

// readLine returns the next line if input and a bool to indicate
// whether the end of the input has been reached.
func (p *parser) readLine() (string, bool) {
	if p.fin {
		return "", true
	}
	if p.storedLine != "" {
		line := p.storedLine
		p.storedLine = ""
		return line, false
	}
	for p.scanner.Scan() {
		line := p.scanner.Text()
		p.lineNum++

		// Check for BOM
		if r, n := utf8.DecodeRuneInString(line); r == 0xfeff || r == 0xfffe {
			line = line[n:]
		}

		// Ignore leading and trailing spaces
		line = strings.TrimSpace(line)

		// TODO: ideally these could be excluded from the exported text to begin with
		line = reAnnotation.ReplaceAllString(line, "")

		// Treat page breaks as empty lines
		if line == "________________" {
			line = ""
		}

		if line != "" && line[0] == '#' {
			continue
		}

		// End of algorithm marker. This is optional but useful to ignore the
		// annotations/commntse google docs adds to the end of the text file.
		if line == "[FIN]" {
			p.fin = true
			return "", true
		}

		return line, false
	}
	p.fin = true
	if err := p.scanner.Err(); err != nil {
		p.err(err.Error())
	}
	return "", true
}

// parseDirectives parses out any [directives] from the line and returns
// the line with them removed
func (p *parser) parseDirectives(line string) (map[string]string, string) {
	directives := make(map[string]string)
	line = reDirective.ReplaceAllStringFunc(line, func(dir string) string {
		dir = dir[1 : len(dir)-1]
		dir = directiveReplacer.Replace(dir)

		if ix := strings.IndexRune(dir, '"'); ix > 0 {
			name := strings.ToLower(strings.TrimSpace(dir[:ix]))
			value, err := strconv.Unquote(strings.TrimSpace(dir[ix:]))
			if err != nil {
				p.err("Bad directive %s: %s", dir, err.Error())
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

func (p *parser) parseSingleDirective(line string) (string, string) {
	directives, _ := p.parseDirectives(line)
	if len(directives) != 1 {
		p.err("Expected a single directive")
	}
	for n, v := range directives {
		return n, v
	}
	panic("shouldn't get here")
}

func (p *parser) err(fm string, args ...interface{}) {
	panic(ParseError{
		Line: p.lineNum,
		Msg:  fmt.Sprintf(fm, args...),
	})
}

func (p *parser) trace(fm string, args ...interface{}) {
	if Debug {
		fmt.Fprintf(os.Stderr, "TRACE %d: %s\n", p.lineNum, fmt.Sprintf(fm, args...))
	}
}
