package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

func loadSpec(t *testing.T) map[string]any {
	t.Helper()
	var spec map[string]any
	if err := json.Unmarshal(openapiJSON, &spec); err != nil {
		t.Fatalf("openapi.json: %v", err)
	}
	return spec
}

func TestSpecCoversRoutes(t *testing.T) {
	spec := loadSpec(t)
	var documented, served []string
	for p := range spec["paths"].(map[string]any) {
		documented = append(documented, p)
	}
	for _, rt := range routes {
		method, path, _ := strings.Cut(rt.pattern, " ")
		if method != http.MethodGet {
			t.Errorf("%s: only GET routes are expected", rt.pattern)
		}
		served = append(served, path)
	}
	sort.Strings(documented)
	sort.Strings(served)
	if !slices.Equal(documented, served) {
		t.Errorf("documented paths %v\nserved paths %v", documented, served)
	}
}

func TestSpecPathParams(t *testing.T) {
	spec := loadSpec(t)
	wildcard := regexp.MustCompile(`\{(\w+)\}`)
	for path, item := range spec["paths"].(map[string]any) {
		get := item.(map[string]any)["get"].(map[string]any)
		var inPath, declared []string
		for _, m := range wildcard.FindAllStringSubmatch(path, -1) {
			inPath = append(inPath, m[1])
		}
		for _, p := range get["parameters"].([]any) {
			if p := p.(map[string]any); p["in"] == "path" {
				declared = append(declared, p["name"].(string))
			}
		}
		if !slices.Equal(inPath, declared) {
			t.Errorf("%s: path has %v, parameters declare %v", path, inPath, declared)
		}
	}
}

func TestSpecRefsResolve(t *testing.T) {
	spec := loadSpec(t)
	schemas := spec["components"].(map[string]any)["schemas"].(map[string]any)
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				name := strings.TrimPrefix(ref, "#/components/schemas/")
				if _, ok := schemas[name]; !ok {
					t.Errorf("unresolved $ref %s", ref)
				}
			}
			for _, x := range v {
				walk(x)
			}
		case []any:
			for _, x := range v {
				walk(x)
			}
		}
	}
	walk(spec)
}

func TestResponsesMatchSpec(t *testing.T) {
	spec := loadSpec(t)
	paths := spec["paths"].(map[string]any)
	schemas := spec["components"].(map[string]any)["schemas"].(map[string]any)
	requests := map[string]string{
		"/counties":                         "/counties",
		"/counties/at":                      "/counties/at?lat=-0.0917&lng=34.768",
		"/counties/{county}":                "/counties/nakuru",
		"/counties/{county}/boundary":       "/counties/47/boundary",
		"/counties/{county}/constituencies": "/counties/22/constituencies",
		"/counties/{county}/neighbours":     "/counties/47/neighbours",
		"/counties/{county}/voters":         "/counties/8/voters",
		"/counties/{county}/wards":          "/counties/mombasa/wards",
		"/counties/{county}/postoffices":    "/counties/47/postoffices",
		"/constituencies/{name}/voters":     "/constituencies/eldas/voters",
		"/constituencies/{name}/wards":      "/constituencies/eldas/wards",
		"/voters":                           "/voters",
		"/national":                         "/national",
		"/wards/{code}":                     "/wards/0181",
		"/wards/{code}/boundary":            "/wards/1439/boundary",
		"/postcodes/{code}":                 "/postcodes/00100",
		"/search":                           "/search?q=kibra&limit=20",
		"/addresses":                        "/addresses?q=P.O.%20Box%20123-00100%20Nairobi",
		"/at":                               "/at?lat=-1.2884&lng=36.8233",
		"/postoffices/near":                 "/postoffices/near?lat=-0.2833&lng=36.0667",
		"/openapi.json":                     "/openapi.json",
	}
	errorRequests := map[string]string{
		"/counties/{county}": "/counties/48",
		"/search":            "/search",
		"/addresses":         "/addresses?q=Box%201-00100%20Mombasa",
	}
	for path := range paths {
		if _, ok := requests[path]; !ok {
			t.Errorf("no sample request for %s", path)
		}
	}

	check := func(path, url string) {
		if paths[path] == nil {
			t.Errorf("%s is not documented", path)
			return
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		responses := paths[path].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)
		resp, ok := responses[fmt.Sprint(rec.Code)].(map[string]any)
		if !ok {
			t.Errorf("%s: status %d is not documented", url, rec.Code)
			return
		}
		content := resp["content"].(map[string]any)
		media, ok := content[rec.Header().Get("Content-Type")].(map[string]any)
		if !ok {
			t.Errorf("%s: Content-Type %q is not documented", url, rec.Header().Get("Content-Type"))
			return
		}
		var body any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Errorf("%s: %v", url, err)
			return
		}
		for _, problem := range validate(body, media["schema"].(map[string]any), schemas, "body") {
			t.Errorf("%s: %s", url, problem)
		}
	}
	for path, url := range requests {
		check(path, url)
	}
	for path, url := range errorRequests {
		check(path, url)
	}
}

func validate(v any, s map[string]any, schemas map[string]any, at string) []string {
	if ref, ok := s["$ref"].(string); ok {
		return validate(v, schemas[strings.TrimPrefix(ref, "#/components/schemas/")].(map[string]any), schemas, at)
	}
	if alts, ok := s["oneOf"].([]any); ok {
		for _, alt := range alts {
			if len(validate(v, alt.(map[string]any), schemas, at)) == 0 {
				return nil
			}
		}
		return []string{at + ": matches none of oneOf"}
	}
	if parts, ok := s["allOf"].([]any); ok {
		merged := map[string]any{"type": "object", "properties": map[string]any{}, "required": []any{}}
		for _, p := range parts {
			p := p.(map[string]any)
			if ref, ok := p["$ref"].(string); ok {
				p = schemas[strings.TrimPrefix(ref, "#/components/schemas/")].(map[string]any)
			}
			for k, x := range p["properties"].(map[string]any) {
				merged["properties"].(map[string]any)[k] = x
			}
			merged["required"] = append(merged["required"].([]any), p["required"].([]any)...)
		}
		return validate(v, merged, schemas, at)
	}
	if c, ok := s["const"]; ok && v != c {
		return []string{fmt.Sprintf("%s: %v, want %v", at, v, c)}
	}
	if enum, ok := s["enum"].([]any); ok && !slices.Contains(enum, v) {
		return []string{fmt.Sprintf("%s: %v is not one of %v", at, v, enum)}
	}

	var problems []string
	switch s["type"] {
	case "object":
		obj, ok := v.(map[string]any)
		if !ok {
			return []string{at + ": not an object"}
		}
		props, _ := s["properties"].(map[string]any)
		if props == nil {
			return nil
		}
		for _, r := range s["required"].([]any) {
			if _, ok := obj[r.(string)]; !ok {
				problems = append(problems, fmt.Sprintf("%s: missing %s", at, r))
			}
		}
		for k, x := range obj {
			ps, ok := props[k].(map[string]any)
			if !ok {
				problems = append(problems, fmt.Sprintf("%s: undocumented field %s", at, k))
				continue
			}
			problems = append(problems, validate(x, ps, schemas, at+"."+k)...)
		}
	case "array":
		arr, ok := v.([]any)
		if !ok {
			return []string{at + ": not an array"}
		}
		items := s["items"].(map[string]any)
		for i, x := range arr {
			problems = append(problems, validate(x, items, schemas, fmt.Sprintf("%s[%d]", at, i))...)
			if len(problems) > 5 {
				break
			}
		}
	case "string":
		if _, ok := v.(string); !ok {
			problems = append(problems, at+": not a string")
		}
	case "number":
		if _, ok := v.(float64); !ok {
			problems = append(problems, at+": not a number")
		}
	case "integer":
		if f, ok := v.(float64); !ok || f != float64(int64(f)) {
			problems = append(problems, at+": not an integer")
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			problems = append(problems, at+": not a boolean")
		}
	case "null":
		if v != nil {
			problems = append(problems, at+": not null")
		}
	}
	return problems
}
