package query

import (
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/scanner"

	"github.com/sprucehealth/backend/errors"
)

/*
-Query Grammar-
Query ::= Expr+
Expr ::= Op[0,1] (PExpr | TExpr)
TExpr ::= ID (Op (PExpr | []TExpr))[0,1]
PExpr ::= `(` Expr `)`
ID ::= (AlphaNumeric+:)*AlphaNumeric
Op ::= ('AND' | 'OR' | 'NOT'| '&' | '|' | '!')
*/

const (
	_ = iota
	And
	Or
	Not
)

var (
	// ErrUnexpectedEOF Represents when the query strign has come to an unexpected end
	ErrUnexpectedEOF = errors.New("Unexpected EOF")
)

type ErrBadExpression interface {
	BadExpressionMessage() string
}

type ErrBadExpr string

func (ber ErrBadExpr) BadExpressionMessage() string {
	return string(ber)
}

func (ber ErrBadExpr) Error() string {
	return string(ber)
}

func IsErrBadExpression(err error) bool {
	_, ok := errors.Cause(err).(ErrBadExpression)
	return ok
}

type TagAssociationQuery struct {
	es []*Expression
}

func NewTagAssociationQuery(q string) (*TagAssociationQuery, error) {
	es, err := scan(q)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &TagAssociationQuery{
		es: es,
	}, nil
}

func (q *TagAssociationQuery) SQL(field string, db *sql.DB) (string, []interface{}, error) {
	m, err := q.tagIDMap(db)
	if err != nil {
		return "", nil, errors.Trace(err)
	}
	sql, v := ExpressionList(q.es).SQL(field, m)
	return `SELECT tag_id, case_id, trigger_time, hidden, created FROM tag_membership WHERE ` + sql, v, nil
}

func (q *TagAssociationQuery) tagIDMap(db *sql.DB) (map[ID]int64, error) {
	idMap := make(map[ID]int64)
	var err error
	for _, e := range q.es {
		if idMap, err = tagIDsForExpression(db, e, idMap); err != nil {
			return nil, errors.Trace(err)
		}
	}
	return idMap, nil
}

func tagIDsForExpression(db *sql.DB, e *Expression, m map[ID]int64) (map[ID]int64, error) {
	var err error
	if e.TE != nil {
		if m, err = tagIDsForTExpr(db, e.TE, m); err != nil {
			return nil, errors.Trace(err)
		}
	}
	if e.PE != nil {
		if m, err = tagIDsForExpression(db, (*Expression)(e.PE), m); err != nil {
			return nil, errors.Trace(err)
		}
	}
	return m, nil
}

func tagIDsForTExpr(db *sql.DB, e *TExpr, m map[ID]int64) (map[ID]int64, error) {
	var err error
	if e.ID != "" {
		if _, ok := m[e.ID]; !ok {
			var id int64
			err := db.QueryRow(`SELECT id FROM tag WHERE tag_text = ?`, string(e.ID)).Scan(&id)
			switch {
			case err == sql.ErrNoRows:
			case err != nil:
				return nil, errors.Trace(err)
			default:
				m[e.ID] = id
			}
		}
	}
	if e.PE != nil {
		if m, err = tagIDsForExpression(db, (*Expression)(e.PE), m); err != nil {
			return nil, errors.Trace(err)
		}
	}
	if e.TE != nil {
		if m, err = tagIDsForTExpr(db, e.TE, m); err != nil {
			return nil, errors.Trace(err)
		}
	}
	return m, nil
}

// Note: This isn't super efficient when it comes to large query strings as it keeps the tokenized info in memory.
// Should be sufficient for this purpose though.
type BulkTokenizer struct {
	sc     scanner.Scanner
	tokens []string
	pos    int
}

type Checkpoint int

func (bt *BulkTokenizer) Checkpoint() Checkpoint {
	return Checkpoint(bt.pos)
}

func (bt *BulkTokenizer) Rewind(c Checkpoint) {
	bt.pos = int(c)
}

func (bt *BulkTokenizer) NextToken() (string, error) {
	if bt.pos == len(bt.tokens) {
		return "", io.EOF
	}
	bt.pos++
	return bt.tokens[bt.pos-1], nil
}

func (bt *BulkTokenizer) Peek() (string, error) {
	if bt.pos == len(bt.tokens) {
		return "", io.EOF
	}
	return bt.tokens[bt.pos], nil
}

func (bt *BulkTokenizer) Tokenize(s string) {
	var sc scanner.Scanner
	sc.Init(strings.NewReader(s))
	bt.sc = sc
	bt.tokens = make([]string, 0)
	for tok := bt.sc.Scan(); tok != scanner.EOF; tok = bt.sc.Scan() {
		bt.tokens = append(bt.tokens, bt.sc.TokenText())
	}
}

func (bt *BulkTokenizer) Pos() int {
	return bt.pos
}

type Op int

func (o Op) String() string {
	switch o {
	case And:
		return `AND`
	case Or:
		return `OR`
	case Not:
		return `NOT`
	}
	return "UNKNOWN OP"
}

func (o Op) SQL(field string) string {
	switch o {
	case And:
		return `AND`
	case Or:
		return `OR`
	case Not:
		return `NOT`
	}
	return "UNKNOWN OP"
}

type ID string

const idScopeOperator = ":"

var idRegex = regexp.MustCompile(`^(\w+:)*\w+$`)

// Validate Asserts that the contents of the type match the expected pattern for an identifier
func (id ID) Validate() error {
	if !idRegex.Match([]byte(id)) {
		return errors.Trace(fmt.Errorf("%s is not a valid identifier", id))
	}
	return nil
}

type PExpr Expression

func (p *PExpr) String() string {
	return `(` + (*Expression)(p).String() + `)`
}

type TExpr struct {
	ID ID
	O  Op
	TE *TExpr
	PE *PExpr
}

func (t *TExpr) String() string {
	s := string(t.ID)
	if t.O != 0 {
		s += t.O.String()
	}
	if t.TE != nil {
		return s + t.TE.String()
	}
	if t.PE != nil {
		return s + t.PE.String()
	}
	return s
}

func (t *TExpr) SQL(field string, m map[ID]int64) (string, []interface{}) {
	v := make([]interface{}, 1)
	s := make([]string, 1, 3)
	id := m[t.ID]
	s[0] = field + ` IN (SELECT case_id FROM tag_membership WHERE tag_id = ?)`
	v[0] = id
	if t.O != 0 {
		s = append(s, t.O.SQL(field))
	}
	if t.TE != nil {
		sr, vr := t.TE.SQL(field, m)
		s = append(s, sr)
		v = append(v, vr...)
	}
	if t.PE != nil {
		sr, vr := (*Expression)(t.PE).SQL(field, m)
		s = append(s, `(`+sr+`)`)
		v = append(v, vr...)
	}
	return strings.Join(s, " "), v
}

type ExpressionList []*Expression

func (el ExpressionList) SQL(field string, m map[ID]int64) (string, []interface{}) {
	s := make([]string, len(([]*Expression)(el)))
	var vs []interface{}
	for i, v := range ([]*Expression)(el) {
		var vr []interface{}
		s[i], vr = v.SQL(field, m)
		vs = append(vs, vr...)
	}
	return strings.Join(s, " "), vs
}

type Expression struct {
	O  Op
	TE *TExpr
	PE *PExpr
}

func (e *Expression) String() string {
	var s string
	if e.O != 0 {
		s += e.O.String()
	}
	if e.TE != nil {
		s += e.TE.String()
	}
	if e.PE != nil {
		s += e.PE.String()
	}
	return s
}

func (p *Expression) SQL(field string, m map[ID]int64) (string, []interface{}) {
	var v []interface{}
	s := make([]string, 0, 3)
	if p.O != 0 {
		sr := p.O.SQL(field)
		s = append(s, sr)
	}
	if p.TE != nil {
		sr, vr := p.TE.SQL(field, m)
		s = append(s, sr)
		v = append(v, vr...)
	}
	if p.PE != nil {
		sr, vr := (*Expression)(p.PE).SQL(field, m)
		s = append(s, `(`+sr+`)`)
		v = append(v, vr...)
	}
	return strings.Join(s, " "), v
}

func scan(s string) ([]*Expression, error) {
	s = strings.TrimSpace(s)
	bt := &BulkTokenizer{}
	bt.Tokenize(s)
	es := make([]*Expression, 0)
	var err error
	_, err = bt.Peek()
	for err != io.EOF {
		var exp *Expression
		exp, err = scanExpression(bt)
		if err != nil {
			return nil, errors.Trace(err)
		}
		es = append(es, exp)
		_, err = bt.Peek()
	}
	return es, nil
}

func scanExpression(s *BulkTokenizer) (*Expression, error) {
	e := &Expression{}
	var err error
	c := s.Checkpoint()
	if e.O, err = scanOp(s); err != nil {
		s.Rewind(c)
	}
	c = s.Checkpoint()
	if e.PE, err = scanPExpr(s); err != nil {
		s.Rewind(c)
		if e.TE, err = scanTExpr(s); err != nil {
			return nil, ErrBadExpr(err.Error())
		}
	}
	return e, nil
}

func scanTExpr(s *BulkTokenizer) (*TExpr, error) {
	te := &TExpr{}
	var err error
	tok, err := s.NextToken()
	if err == io.EOF {
		return nil, errors.Trace(ErrUnexpectedEOF)
	}
	idtok, err := s.Peek()
	for idtok == idScopeOperator {
		t, err := s.NextToken()
		if err == io.EOF {
			return nil, errors.Trace(ErrUnexpectedEOF)
		}
		tok += t
		t, err = s.NextToken()
		if err == io.EOF {
			return nil, errors.Trace(ErrUnexpectedEOF)
		}
		tok += t
		idtok, err = s.Peek()
	}
	te.ID = ID(tok)
	if err := te.ID.Validate(); err != nil {
		return nil, errors.Trace(err)
	}
	t, err := s.Peek()
	if err == io.EOF || t == ")" {
		return te, nil
	}
	if te.O, err = scanOp(s); err != nil {
		return nil, errors.Trace(err)
	}
	if te.O == Not {
		return nil, errors.New("Cannot use NOT as an infix operator.")
	}
	c := s.Checkpoint()
	if te.PE, err = scanPExpr(s); err != nil {
		s.Rewind(c)
		if te.TE, err = scanTExpr(s); err != nil {
			return nil, err
		}
	}
	return te, nil
}

func scanPExpr(s *BulkTokenizer) (*PExpr, error) {
	tok, err := s.NextToken()
	if err == io.EOF {
		return nil, errors.Trace(ErrUnexpectedEOF)
	}
	if tok != "(" {
		return nil, fmt.Errorf("Expected '(' but found %s", tok)
	}
	e, err := scanExpression(s)
	if err != nil {
		return nil, err
	}
	tok, err = s.NextToken()
	if err == io.EOF {
		return nil, errors.Trace(ErrUnexpectedEOF)
	}
	if tok != ")" {
		return nil, fmt.Errorf("Expected ')' but found %s", tok)
	}
	return (*PExpr)(e), nil
}

func scanOp(s *BulkTokenizer) (Op, error) {
	tok, err := s.NextToken()
	if err == io.EOF {
		return 0, errors.Trace(ErrUnexpectedEOF)
	}
	switch strings.ToLower(tok) {
	case `and`, `&`:
		return And, nil
	case `or`, `|`:
		return Or, nil
	case `not`, `!`:
		return Not, nil
	}
	return 0, fmt.Errorf("Unknown operator %s", tok)
}
