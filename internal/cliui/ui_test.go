package cliui

import (
	"bytes"
	"strings"
	"testing"
)

func TestLineNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	Line(&buf, LevelOK, "done")
	if !strings.Contains(buf.String(), "ok done") {
		t.Fatalf("got %q", buf.String())
	}
}

func TestPrintKV(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	PrintKV(&buf, [][2]string{{"version", "v1.0.0"}, {"platform", "linux/amd64"}})
	out := buf.String()
	if !strings.Contains(out, "version:") || !strings.Contains(out, "v1.0.0") {
		t.Fatalf("got %q", out)
	}
}

func TestPrintTable(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	PrintTable(&buf, []string{"ID", "ALIAS"}, [][]string{{"abc", "x"}})
	if !strings.Contains(buf.String(), "ID") || !strings.Contains(buf.String(), "abc") {
		t.Fatalf("got %q", buf.String())
	}
}
