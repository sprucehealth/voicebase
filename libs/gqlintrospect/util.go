package gqlintrospect

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

func printUnion(w io.Writer, t *Type) error {
	if t.Description != nil && *t.Description != "" {
		if _, err := fmt.Fprintf(w, "// %s\n", *t.Description); err != nil {
			return err
		}
	}
	sort.Sort(typesName(t.PossibleTypes))
	if _, err := fmt.Fprintf(w, "union %s = ", *t.Name); err != nil {
		return err
	}
	for i, t := range t.PossibleTypes {
		if i != 0 {
			if _, err := fmt.Fprint(w, " | "); err != nil {
				return err
			}
		}
		tn, err := typeName(t)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, tn); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return nil
}

func printEnum(w io.Writer, t *Type) error {
	sort.Sort(enumsName(t.EnumValues))
	if t.Description != nil && *t.Description != "" {
		if _, err := fmt.Fprintf(w, "// %s\n", *t.Description); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "enum %s {\n", *t.Name); err != nil {
		return err
	}
	for _, e := range t.EnumValues {
		if e.IsDeprecated {
			if err := printDeprecation(w, e.DeprecationReason); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, "\t", e.Name); err != nil {
			return err
		}
		if e.Description != nil && *e.Description != "" {
			if _, err := fmt.Fprintf(w, "\t// %s\n", *e.Description); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(w, "}"); err != nil {
		return err
	}
	return nil
}

func printObject(w io.Writer, t *Type) error {
	if t.Description != nil && *t.Description != "" {
		if _, err := fmt.Fprintf(w, "// %s\n", *t.Description); err != nil {
			return err
		}
	}
	sort.Sort(typesName(t.Interfaces))
	if _, err := fmt.Fprintf(w, "type %s", *t.Name); err != nil {
		return err
	}
	for _, in := range t.Interfaces {
		if _, err := fmt.Fprintf(w, " : %s", *in.Name); err != nil {
			return err
		}
	}
	sort.Sort(fieldsName(t.Fields))
	if _, err := fmt.Fprintln(w, " {"); err != nil {
		return err
	}
	for _, f := range t.Fields {
		if f.IsDeprecated {
			if err := printDeprecation(w, f.DeprecationReason); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "\t%s", f.Name); err != nil {
			return err
		}
		if err := printArgs(w, f.Args); err != nil {
			return err
		}
		tn, err := typeName(f.Type)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, ": %s", tn); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "}"); err != nil {
		return err
	}
	return nil
}

func printInputObject(w io.Writer, t *Type) error {
	if t.Description != nil && *t.Description != "" {
		if _, err := fmt.Fprintf(w, "// %s\n", *t.Description); err != nil {
			return err
		}
	}
	sort.Sort(inputValuesName(t.InputFields))
	if _, err := fmt.Fprintf(w, "input %s {\n", *t.Name); err != nil {
		return err
	}
	for _, f := range t.InputFields {
		tn, err := typeName(f.Type)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "\t%s: %s", f.Name, tn); err != nil {
			return err
		}
		if f.DefaultValue != nil && *f.DefaultValue != "" {
			if _, err := fmt.Fprintf(w, " = %s", strconv.Quote(*f.DefaultValue)); err != nil {
				return err
			}
		}
		if f.Description != nil && *f.Description != "" {
			if _, err := fmt.Fprintf(w, "\t// %s", *f.Description); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "}"); err != nil {
		return err
	}
	return nil
}

func printInterface(w io.Writer, t *Type) error {
	if t.Description != nil && *t.Description != "" {
		if _, err := fmt.Fprintf(w, "// %s\n", *t.Description); err != nil {
			return err
		}
	}
	if len(t.PossibleTypes) != 0 {
		if _, err := fmt.Fprintf(w, "// Implemented by types:"); err != nil {
			return err
		}
		for _, pt := range t.PossibleTypes {
			if _, err := fmt.Fprintf(w, " %s", *pt.Name); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	sort.Sort(fieldsName(t.Fields))
	if _, err := fmt.Fprintf(w, "interface %s {\n", *t.Name); err != nil {
		return err
	}
	for _, f := range t.Fields {
		if f.IsDeprecated {
			if err := printDeprecation(w, f.DeprecationReason); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "\t%s", f.Name); err != nil {
			return err
		}
		if err := printArgs(w, f.Args); err != nil {
			return err
		}
		tn, err := typeName(f.Type)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, ": %s", tn); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "}"); err != nil {
		return err
	}
	return nil
}

func printDeprecation(w io.Writer, reason *string) error {
	var err error
	if reason != nil && *reason != "" {
		_, err = fmt.Fprintf(w, "\t// DEPRECATED: %s\n", *reason)
	} else {
		_, err = fmt.Fprintln(w, "\t// DEPRECATED")
	}
	return err
}

func printArgs(w io.Writer, args []*InputValue) error {
	if len(args) == 0 {
		return nil
	}
	sort.Sort(inputValuesName(args))
	as := make([]string, len(args))
	for i, a := range args {
		tn, err := typeName(a.Type)
		if err != nil {
			return err
		}
		as[i] = fmt.Sprintf("%s: %s", a.Name, tn)
		if a.DefaultValue != nil && *a.DefaultValue != "" {
			as[i] += fmt.Sprintf(" = %s", strconv.Quote(*a.DefaultValue))
		}
		if a.Description != nil && *a.Description != "" {
			as[i] += fmt.Sprintf(" /* %s */", *a.Description)
		}
	}
	if _, err := fmt.Fprintf(w, "(%s)", strings.Join(as, ", ")); err != nil {
		return err
	}
	return nil
}

func typeName(t *Type) (string, error) {
	if t.Name != nil && *t.Name != "" {
		return *t.Name, nil
	}
	switch t.Kind {
	case NonNull:
		tn, err := typeName(t.OfType)
		if err != nil {
			return "", err
		}
		return tn + "!", nil
	case List:
		tn, err := typeName(t.OfType)
		if err != nil {
			return "", err
		}
		return "[" + tn + "]", nil
	}
	return "", fmt.Errorf("Unable to resolve name of type %+v", t)
}

type typesName []*Type

func (n typesName) Len() int           { return len(n) }
func (n typesName) Swap(a, b int)      { n[a], n[b] = n[b], n[a] }
func (n typesName) Less(a, b int) bool { return lessPtrStrLess(n[a].Name, n[b].Name) }

type fieldsName []*Field

func (n fieldsName) Len() int           { return len(n) }
func (n fieldsName) Swap(a, b int)      { n[a], n[b] = n[b], n[a] }
func (n fieldsName) Less(a, b int) bool { return n[a].Name < n[b].Name }

type enumsName []*EnumValue

func (n enumsName) Len() int           { return len(n) }
func (n enumsName) Swap(a, b int)      { n[a], n[b] = n[b], n[a] }
func (n enumsName) Less(a, b int) bool { return n[a].Name < n[b].Name }

type inputValuesName []*InputValue

func (n inputValuesName) Len() int           { return len(n) }
func (n inputValuesName) Swap(a, b int)      { n[a], n[b] = n[b], n[a] }
func (n inputValuesName) Less(a, b int) bool { return n[a].Name < n[b].Name }

func lessPtrStrLess(a, b *string) bool {
	if b == nil {
		return true
	}
	if a == nil {
		return false
	}
	return *a < *b
}
