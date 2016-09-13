package threading

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestQueries(t *testing.T) {
	cases := []*struct {
		s string
		q *Query
	}{
		{s: "", q: &Query{Expressions: []*Expr{}}},
		{s: "foo", q: &Query{Expressions: []*Expr{{Value: &Expr_Token{Token: "foo"}}}}},
		{s: "is:unread", q: &Query{Expressions: []*Expr{{Value: &Expr_Flag_{Flag: EXPR_FLAG_UNREAD}}}}},
		{s: "is:read", q: &Query{Expressions: []*Expr{{Not: true, Value: &Expr_Flag_{Flag: EXPR_FLAG_UNREAD}}}}},
		{s: "is:following", q: &Query{Expressions: []*Expr{{Value: &Expr_Flag_{Flag: EXPR_FLAG_FOLLOWING}}}}},
		{s: "type:patient", q: &Query{Expressions: []*Expr{{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_PATIENT}}}}},
		{s: "type:team", q: &Query{Expressions: []*Expr{{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_TEAM}}}}},
		{s: "type:patient is:unreadreference", q: &Query{Expressions: []*Expr{
			{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_PATIENT}},
			{Value: &Expr_Flag_{Flag: EXPR_FLAG_UNREAD_REFERENCE}},
		}}},
		{s: "type:patient Esther Smith", q: &Query{Expressions: []*Expr{
			{Value: &Expr_ThreadType_{ThreadType: EXPR_THREAD_TYPE_PATIENT}},
			{Value: &Expr_Token{Token: "Esther"}},
			{Value: &Expr_Token{Token: "Smith"}},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.s, func(t *testing.T) {
			a, err := ParseQuery(tc.s)
			test.OK(t, err)
			test.Equals(t, tc.q, a)
			f, err := FormatQuery(a)
			test.OK(t, err)
			test.Equals(t, tc.s, f)
		})
	}
}
