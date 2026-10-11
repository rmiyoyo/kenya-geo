package api

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	kenyageo "github.com/rmiyoyo/kenya-geo"
)

func TestDocsCoverRoutes(t *testing.T) {
	var documented, served []string
	for _, m := range regexp.MustCompile(`(?m)^### (GET /\S*)\r?$`).FindAllStringSubmatch(Docs, -1) {
		documented = append(documented, m[1])
	}
	for _, rt := range routes {
		served = append(served, rt.pattern)
	}
	sort.Strings(documented)
	sort.Strings(served)
	if !slices.Equal(documented, served) {
		t.Errorf("documented %v\nserved %v", documented, served)
	}
}

func TestDocsExamplesWork(t *testing.T) {
	h := New(kenyageo.Default())
	examples := regexp.MustCompile("(?m)^```http\\r?\\nGET (\\S+)\\r?\\n```\\r?$").FindAllStringSubmatch(Docs, -1)
	if len(examples) < len(routes) {
		t.Errorf("%d examples for %d routes", len(examples), len(routes))
	}
	for _, m := range examples {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, m[1], nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status %d: %s", m[1], rec.Code, strings.TrimSpace(rec.Body.String()))
		}
	}
}
