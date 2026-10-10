package api

import (
	"os"
	"strings"
	"testing"
)

func TestDocsListEveryRoute(t *testing.T) {
	b, err := os.ReadFile("../docs/API.md")
	if err != nil {
		t.Fatal(err)
	}
	headings := map[string]bool{}
	for _, line := range strings.Split(string(b), "\n") {
		if h, ok := strings.CutPrefix(line, "### "); ok {
			headings[h] = true
		}
	}
	for _, rt := range routes {
		if !headings[rt.pattern] {
			t.Errorf("docs/API.md has no %q heading", "### "+rt.pattern)
		}
		delete(headings, rt.pattern)
	}
	for h := range headings {
		t.Errorf("docs/API.md documents %q, which the API doesn't serve", h)
	}
}
