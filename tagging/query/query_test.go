package query

import (
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestTaggingQueryScanComplexQueries(t *testing.T) {
	testSets := []struct {
		input    string
		expected []*Expression
	}{
		{`A`, []*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
				},
			},
		}},
		{`(A)`, []*Expression{
			&Expression{
				PE: &PExpr{
					TE: &TExpr{
						ID: ID(`A`),
					},
				},
			},
		}},
		{`A AND B`, []*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					TE: &TExpr{
						ID: ID(`B`),
					},
				},
			},
		}},
		{`A & B`, []*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					TE: &TExpr{
						ID: ID(`B`),
					},
				},
			},
		}},
		{`A OR B`, []*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  Or,
					TE: &TExpr{
						ID: ID(`B`),
					},
				},
			},
		}},
		{`A | B`, []*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  Or,
					TE: &TExpr{
						ID: ID(`B`),
					},
				},
			},
		}},
		{`!B`, []*Expression{
			&Expression{
				O: Not,
				TE: &TExpr{
					ID: ID(`B`),
				},
			},
		}},
		{`A & (NOT B)`, []*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					PE: &PExpr{
						O: Not,
						TE: &TExpr{
							ID: ID(`B`),
						},
					},
				},
			},
		}},
		{`A & (B OR C | D)`, []*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					PE: &PExpr{
						TE: &TExpr{
							ID: ID(`B`),
							O:  Or,
							TE: &TExpr{
								ID: ID(`C`),
								O:  Or,
								TE: &TExpr{
									ID: ID(`D`),
								},
							},
						},
					},
				},
			},
		}},
		{`(A OR B) AND (NOT C)`, []*Expression{
			&Expression{
				PE: &PExpr{
					TE: &TExpr{
						ID: ID(`A`),
						O:  Or,
						TE: &TExpr{
							ID: ID(`B`),
						},
					},
				},
			},
			&Expression{
				O: And,
				PE: &PExpr{
					O: Not,
					TE: &TExpr{
						ID: ID(`C`),
					},
				},
			},
		}},
		{`(A OR B AND (NOT D)) AND (NOT C)`, []*Expression{
			&Expression{
				PE: &PExpr{
					TE: &TExpr{
						ID: ID(`A`),
						O:  Or,
						TE: &TExpr{
							ID: ID(`B`),
							O:  And,
							PE: &PExpr{
								O: Not,
								TE: &TExpr{
									ID: ID(`D`),
								},
							},
						},
					},
				},
			},
			&Expression{
				O: And,
				PE: &PExpr{
					O: Not,
					TE: &TExpr{
						ID: ID(`C`),
					},
				},
			},
		}},
	}

	for _, v := range testSets {
		fmt.Println(v.input)
		es, err := scan(v.input)
		test.OK(t, err)
		test.Equals(t, v.expected, es)
	}
}

func TestTaggingQueryScanError(t *testing.T) {
	testSets := []struct {
		input       string
		expectedErr error
	}{
		{`A NOT B`, ErrBadExpression},
	}
	for _, v := range testSets {
		fmt.Println(v.input)
		_, err := scan(v.input)
		test.Equals(t, v.expectedErr, err)
	}
}

func TestTaggingQuerySQLGeneration(t *testing.T) {
	testSets := []struct {
		input     []*Expression
		m         map[ID]int64
		field     string
		expectedQ string
		expectedV []interface{}
	}{
		{[]*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
		},
			`case_id`,
			`case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?)`,
			[]interface{}{
				int64(1),
			},
		},
		{[]*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  Or,
					TE: &TExpr{
						ID: ID(`B`),
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
			ID(`B`): 2,
		},
			`case_id`,
			`case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) OR case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?)`,
			[]interface{}{
				int64(1),
				int64(2),
			},
		},
		{[]*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					TE: &TExpr{
						ID: ID(`B`),
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
			ID(`B`): 2,
		},
			`case_id`,
			`case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) AND case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?)`,
			[]interface{}{
				int64(1),
				int64(2),
			},
		},
		{[]*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					PE: &PExpr{
						TE: &TExpr{
							ID: ID(`B`),
						},
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
			ID(`B`): 2,
		},
			`case_id`,
			`case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) AND (case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?))`,
			[]interface{}{
				int64(1),
				int64(2),
			},
		},
		{[]*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					PE: &PExpr{
						TE: &TExpr{
							ID: ID(`B`),
							O:  Or,
							TE: &TExpr{
								ID: ID(`C`),
							},
						},
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
			ID(`B`): 2,
			ID(`C`): 3,
		},
			`case_id`,
			`case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) AND (case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) OR case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?))`,
			[]interface{}{
				int64(1),
				int64(2),
				int64(3),
			},
		},
		{[]*Expression{
			&Expression{
				PE: &PExpr{
					TE: &TExpr{
						ID: ID(`A`),
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
		},
			`case_id`,
			`(case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?))`,
			[]interface{}{
				int64(1),
			},
		},
		{[]*Expression{
			&Expression{
				O: Not,
				TE: &TExpr{
					ID: ID(`B`),
				},
			},
		}, map[ID]int64{
			ID(`B`): 1,
		},
			`case_id`,
			`NOT case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?)`,
			[]interface{}{
				int64(1),
			},
		},
		{[]*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					PE: &PExpr{
						O: Not,
						TE: &TExpr{
							ID: ID(`B`),
						},
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
			ID(`B`): 2,
		},
			`case_id`,
			`case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) AND (NOT case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?))`,
			[]interface{}{
				int64(1),
				int64(2),
			},
		},
		{[]*Expression{
			&Expression{
				TE: &TExpr{
					ID: ID(`A`),
					O:  And,
					PE: &PExpr{
						TE: &TExpr{
							ID: ID(`B`),
							O:  Or,
							TE: &TExpr{
								ID: ID(`C`),
								O:  Or,
								TE: &TExpr{
									ID: ID(`D`),
								},
							},
						},
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
			ID(`B`): 2,
			ID(`C`): 3,
			ID(`D`): 4,
		},
			`case_id`,
			`case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) AND (case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) OR case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) OR case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?))`,
			[]interface{}{
				int64(1),
				int64(2),
				int64(3),
				int64(4),
			},
		},
		{[]*Expression{
			&Expression{
				PE: &PExpr{
					TE: &TExpr{
						ID: ID(`A`),
						O:  Or,
						TE: &TExpr{
							ID: ID(`B`),
						},
					},
				},
			},
			&Expression{
				O: And,
				PE: &PExpr{
					O: Not,
					TE: &TExpr{
						ID: ID(`C`),
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
			ID(`B`): 2,
			ID(`C`): 3,
		},
			`case_id`,
			`(case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) OR case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?)) AND (NOT case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?))`,
			[]interface{}{
				int64(1),
				int64(2),
				int64(3),
			},
		},
		{[]*Expression{
			&Expression{
				PE: &PExpr{
					TE: &TExpr{
						ID: ID(`A`),
						O:  Or,
						TE: &TExpr{
							ID: ID(`B`),
							O:  And,
							PE: &PExpr{
								O: Not,
								TE: &TExpr{
									ID: ID(`D`),
								},
							},
						},
					},
				},
			},
			&Expression{
				O: And,
				PE: &PExpr{
					O: Not,
					TE: &TExpr{
						ID: ID(`C`),
					},
				},
			},
		}, map[ID]int64{
			ID(`A`): 1,
			ID(`B`): 2,
			ID(`C`): 3,
			ID(`D`): 4,
		},
			`case_id`,
			`(case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) OR case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?) AND (NOT case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?))) AND (NOT case_id IN (SELECT case_id FROM tag_membership WHERE tag_id = ?))`,
			[]interface{}{
				int64(1),
				int64(2),
				int64(4),
				int64(3),
			},
		},
	}

	for _, v := range testSets {
		sql, vr := ExpressionList(v.input).SQL(v.field, v.m)
		fmt.Println(sql, vr)
		test.Equals(t, v.expectedQ, sql)
		test.Equals(t, v.expectedV, vr)
	}
}
