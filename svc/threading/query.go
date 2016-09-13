package threading

import (
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
)

// ParseQuery parses a query string into a structure form that can be used
// with the threading service.
func ParseQuery(qs string) (*Query, error) {
	// TODO: Make this parser better. It is a very simple for now and lacks
	//       feature like like quoting of grouped words/tokens.
	parts := strings.Split(qs, " ")
	exprs := make([]*Expr, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var e *Expr
		if ix := strings.IndexRune(p, ':'); ix > 0 {
			key := strings.ToLower(p[:ix])
			value := p[ix+1:]
			switch key {
			case "is": // Flag
				switch value {
				case "unread":
					e = &Expr{Value: &Expr_Flag_{Flag: EXPR_FLAG_UNREAD}}
				case "read":
					e = &Expr{Not: true, Value: &Expr_Flag_{Flag: EXPR_FLAG_UNREAD}}
				case "unreadreference":
					e = &Expr{Value: &Expr_Flag_{Flag: EXPR_FLAG_UNREAD_REFERENCE}}
				case "following":
					e = &Expr{Value: &Expr_Flag_{Flag: EXPR_FLAG_FOLLOWING}}
				}
			case "type": // Type of thread
				switch value {
				case "patient":
					e = &Expr{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_PATIENT}}
				case "team":
					e = &Expr{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_TEAM}}
				}
			}
		}
		if e == nil {
			e = &Expr{Value: &Expr_Token{Token: p}}
		}
		exprs = append(exprs, e)
	}
	return &Query{Expressions: exprs}, nil
}

// FormatQuery returns the textual version of a structured query.
func FormatQuery(q *Query) (string, error) {
	parts := make([]string, 0, len(q.Expressions))
	for _, e := range q.Expressions {
		var not string
		if e.Not {
			not = "-"
		}
		switch v := e.Value.(type) {
		case *Expr_Flag_:
			switch v.Flag {
			case EXPR_FLAG_UNREAD:
				if e.Not {
					parts = append(parts, "is:read")
				} else {
					parts = append(parts, "is:unread")
				}
			case EXPR_FLAG_UNREAD_REFERENCE:
				parts = append(parts, not+"is:unreadreference")
			case EXPR_FLAG_FOLLOWING:
				parts = append(parts, not+"is:following")
			default:
				return "", errors.Errorf("unknown expression flag %s", v.Flag)
			}
		case *Expr_ThreadType_:
			switch v.ThreadType {
			case EXPR_THREAD_TYPE_PATIENT:
				parts = append(parts, not+"type:patient")
			case EXPR_THREAD_TYPE_TEAM:
				parts = append(parts, not+"type:team")
			default:
				return "", errors.Errorf("unknown expression thread type %s", v.ThreadType)
			}
		case *Expr_Token:
			parts = append(parts, not+v.Token)
		default:
			return "", errors.Errorf("unknown expression type %T", e.Value)
		}
	}
	return strings.Join(parts, " "), nil
}
