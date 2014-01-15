package golog

import (
	"testing"
)

func TestBasic(t *testing.T) {
	if GetAppName() != "golog" {
		t.Fatal("Failed to set app name. Expected golog, got %s", GetAppName())
	}
}
