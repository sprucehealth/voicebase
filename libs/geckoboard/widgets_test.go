package geckoboard

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestTextWidget(t *testing.T) {
	tw := &Text{}
	if err := tw.AppendData([]string{"text", "type"}, []interface{}{"text", 1}); err != nil {
		t.Fatal(err)
	} else if len(tw.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(tw.Items))
	} else if tw.Items[0].Text != "text" {
		t.Fatalf("Expected 'text' for item text, got '%s'", tw.Items[0].Text)
	} else if tw.Items[0].Type != AlertItem {
		t.Fatalf("Expected %d for item type, got %d", AlertItem, tw.Items[0].Type)
	}

	tw = &Text{}
	if err := tw.AppendData([]string{"text", "type"}, []interface{}{"foo", "info"}); err != nil {
		t.Fatal(err)
	} else if len(tw.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(tw.Items))
	} else if tw.Items[0].Text != "foo" {
		t.Fatalf("Expected 'foo' for item text, got '%s'", tw.Items[0].Text)
	} else if tw.Items[0].Type != InfoItem {
		t.Fatalf("Expected %d for item type, got %d", InfoItem, tw.Items[0].Type)
	}
}

func TestLeaderboardWidget(t *testing.T) {
	w := &Leaderboard{}
	err := w.AppendData([]string{"label"}, []interface{}{"blah"})
	test.OK(t, err)
	test.Equals(t, 1, len(w.Items))
	test.Equals(t, "blah", w.Items[0].Label)

	w = &Leaderboard{}
	err = w.AppendData([]string{"label", "value"}, []interface{}{"blah", 5.2})
	test.OK(t, err)
	test.Equals(t, 1, len(w.Items))
	test.Equals(t, 5.2, w.Items[0].Value)

	w = &Leaderboard{}
	err = w.AppendData([]string{"label", "value"}, []interface{}{"blah", "foo"})
	test.OK(t, err)
	test.Equals(t, 1, len(w.Items))
	test.Equals(t, "foo", w.Items[0].Value)
}
