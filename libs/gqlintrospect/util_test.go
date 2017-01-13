package gqlintrospect

import (
	"bytes"
	"testing"

	"github.com/sprucehealth/backend/libs/ptr"
)

func TestPrintObject(t *testing.T) {
	obj := &Type{
		Name:        ptr.String("TheName"),
		Description: ptr.String("The description"),
		Interfaces: []*Type{
			{Name: ptr.String("Iface1")},
			{Name: ptr.String("Iface2")},
		},
		Fields: []*Field{
			{Name: "Field1", Type: &Type{Name: ptr.String("Field1Type")}},
		},
	}
	b := &bytes.Buffer{}
	if err := printObject(b, obj); err != nil {
		t.Fatal(err)
	}
	exp := `# The description
type TheName implements Iface1, Iface2 {
	Field1: Field1Type
}
`
	if s := b.String(); s != exp {
		t.Errorf("printObject(b, %#+v) = %q, expected %q", obj, s, exp)
	}
}

func TestPrintDeprecation(t *testing.T) {
	// First test nil case
	b := &bytes.Buffer{}
	if err := printDeprecation(b, nil); err != nil {
		t.Fatal(err)
	}
	if s, e := b.String(), "\t# DEPRECATED\n"; s != e {
		t.Errorf("printDeprecation(b, nil) = %q, expected %q", s, e)
	}

	cases := []struct {
		Reason   string
		Expected string
	}{
		{"", "\t# DEPRECATED\n"},
		{"Short", "\t# DEPRECATED: Short\n"},
		{
			"Really long line that should wrap one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen\nseventeen eighteen",
			`	# DEPRECATED: Really long line that should wrap one two three four five six seven eight nine ten eleven twelve
	#             thirteen fourteen fifteen sixteen
	#             seventeen eighteen
`,
		},
	}
	for _, c := range cases {
		t.Run(c.Reason, func(t *testing.T) {
			b.Reset()
			if err := printDeprecation(b, &c.Reason); err != nil {
				t.Fatal(err)
			}
			if s := b.String(); s != c.Expected {
				t.Errorf("printDeprecation(b, %q) = %q, expected %q", c.Reason, s, c.Expected)
			}
		})
	}
}
