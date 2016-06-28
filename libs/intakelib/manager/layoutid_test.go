package manager

import "testing"

func TestLayoutIDGeneration(t *testing.T) {
	l := layoutUnitID{
		sectionIndex:  1,
		screenIndex:   2,
		questionIndex: 3,
	}

	testLayoutUnitID(t, l, "se:1|sc:2|qu:3")

	l = layoutUnitID{
		sectionIndex: 1,
	}
	testLayoutUnitID(t, l, "se:1")

	l = layoutUnitID{
		sectionIndex: 1,
		screenIndex:  2,
	}
	testLayoutUnitID(t, l, "se:1|sc:2")

	l = layoutUnitID{
		sectionIndex:    1,
		screenIndex:     2,
		questionIndex:   3,
		subquestionInfo: "21409735",
	}
	testLayoutUnitID(t, l, "se:1|sc:2|qu:3|te:21409735")

	l = layoutUnitID{
		sectionIndex:    1,
		screenIndex:     2,
		questionIndex:   3,
		subquestionInfo: "21409735|sc:0",
	}

	testLayoutUnitID(t, l, "se:1|sc:2|qu:3|te:21409735|sc:0")
}

func TestLayoutIDParsing(t *testing.T) {
	l, err := parseLayoutUnitID("se:1|sc:2|qu:3")
	if err != nil {
		t.Fatal(err)
	} else if l.sectionIndex != 1 {
		t.Fatalf("Expected sectionIndex %d but got %d", 1, l.sectionIndex)
	} else if l.screenIndex != 2 {
		t.Fatalf("Expected screenIndex %d but got %d", 2, l.screenIndex)
	} else if l.questionIndex != 3 {
		t.Fatalf("Expected questionIndex %d but got %d", 3, l.questionIndex)
	}

	l, err = parseLayoutUnitID("se:1")
	if err != nil {
		t.Fatal(err)
	} else if l.sectionIndex != 1 {
		t.Fatalf("Expected sectionIndex %d but got %d", 1, l.sectionIndex)
	}

	l, err = parseLayoutUnitID("se:1|sc:2")
	if err != nil {
		t.Fatal(err)
	} else if l.sectionIndex != 1 {
		t.Fatalf("Expected sectionIndex %d but got %d", 1, l.sectionIndex)
	} else if l.screenIndex != 2 {
		t.Fatalf("Expected screenIndex %d but got %d", 2, l.screenIndex)
	}

	l, err = parseLayoutUnitID("se:1|sc:2|qu:3|te:10")
	if err != nil {
		t.Fatal(err)
	} else if l.sectionIndex != 1 {
		t.Fatalf("Expected sectionIndex %d but got %d", 1, l.sectionIndex)
	} else if l.screenIndex != 2 {
		t.Fatalf("Expected screenIndex %d but got %d", 2, l.screenIndex)
	} else if l.questionIndex != 3 {
		t.Fatalf("Expected questionIndex %d but got %d", 3, l.questionIndex)
	} else if l.subquestionInfo != "10" {
		t.Fatal("Expected subquestion index to be populated but it wasnt")
	}

	l, err = parseLayoutUnitID("se:1|sc:2|qu:3|te:1234556|sc:0|qu:0")
	if err != nil {
		t.Fatal(err)
	}

	// this should fail
	_, err = parseLayoutUnitID("")
	if err == nil {
		t.Fatal("Expected parsing empty layoutUnitID to fail but didn't")
	}

	_, err = parseLayoutUnitID("sc:2|se:1|qu:3|an:10|te:159715")
	if err == nil {
		t.Fatal("Expected parsing the layoutUnitiD to error out but it didnt.")
	}
	_, err = parseLayoutUnitID("sc:2|se:1")
	if err == nil {
		t.Fatal("Expected parsing the layoutUnitiD to error out but it didnt.")
	}

	_, err = parseLayoutUnitID("se:1|sc:2|an:10|te:105861")
	if err == nil {
		t.Fatal("Expected parsing the layoutUnitiD to error out but it didnt.")
	}

	_, err = parseLayoutUnitID("te:ha|dlkgan:|se:1|sc:2|an:10")
	if err == nil {
		t.Fatal("Expected parsing the layoutUnitiD to error out but it didnt.")
	}

	_, err = parseLayoutUnitID("se:1|sc:2|qu:3|an:10|te:1234sf556")
	if err == nil {
		t.Fatal("Expected parsing the layoutUnitiD to error out but it didnt.")
	}

}

func BenchmarkLayoutIDParsing(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := parseLayoutUnitID("se:1|sc:2|qu:3|an:10|te:19273586")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLayoutIDGeneration(b *testing.B) {
	l := layoutUnitID{
		sectionIndex:    1,
		screenIndex:     2,
		questionIndex:   3,
		subquestionInfo: "dagkhiag",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = l.String()
	}
}

func testLayoutUnitID(t *testing.T, l layoutUnitID, expected string) {
	if l.String() != expected {
		t.Fatalf("Expected %s, got %s", expected, l.String())
	}
}
