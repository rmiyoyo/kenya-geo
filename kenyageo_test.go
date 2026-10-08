package kenyageo

import (
	"fmt"
	"strings"
	"testing"
)

func TestEmbeddedDataLoads(t *testing.T) {
	d := Default()
	if got := len(d.Counties()); got != 47 {
		t.Errorf("counties = %d, want 47", got)
	}
	if got := len(d.wards); got != 1450 {
		t.Errorf("wards = %d, want 1450", got)
	}
	constituencies := 0
	for _, c := range d.Counties() {
		constituencies += len(d.Constituencies(c.Code))
	}
	// Kenya has 290 constituencies, but the source file labels Gatundu
	// North's four wards (0555-0558) as Gatundu South, so only 289 appear.
	// Update this to 290 once the data is fixed.
	if constituencies != 289 {
		t.Errorf("constituencies = %d, want 289", constituencies)
	}
}

// A table-driven test: one slice of cases, one loop. t.Run gives each case
// its own name in the output, e.g. TestCountyByName/curly_apostrophe.
func TestCountyByName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // county code; 0 means "not found"
	}{
		{"exact", "Nairobi County", 47},
		{"no suffix, lower case", "nairobi", 47},
		{"curly apostrophe", "Murang’a", 21},
		{"straight apostrophe", "murang'a county", 21},
		{"no apostrophe", "Muranga", 21},
		{"hyphen for en dash", "Taita-Taveta", 6},
		{"space for en dash", "taita taveta county", 6},
		{"extra whitespace", "  Homa   Bay ", 43},
		{"hyphenated", "Tharaka Nithi", 13},
		{"unknown", "Atlantis", 0},
		{"empty", "", 0},
	}
	d := Default()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, ok := d.CountyByName(tt.input)
			if tt.want == 0 {
				if ok {
					t.Fatalf("CountyByName(%q) = %v, want not found", tt.input, c.Name)
				}
				return
			}
			if !ok || c.Code != tt.want {
				t.Fatalf("CountyByName(%q) = (%d, %v), want %d", tt.input, c.Code, ok, tt.want)
			}
		})
	}
}

func TestCountyByCode(t *testing.T) {
	d := Default()
	c, ok := d.CountyByCode(1)
	if !ok || c.Name != "Mombasa County" || c.Headquarters != "Mombasa" {
		t.Errorf("CountyByCode(1) = %+v, %v", c, ok)
	}
	if _, ok := d.CountyByCode(48); ok {
		t.Error("CountyByCode(48) found a county")
	}
}

func TestWards(t *testing.T) {
	d := Default()

	w, ok := d.WardByCode("0040")
	if !ok || w.Name != "WAA" || w.CountyCode != 2 || w.Constituency != "Matuga" {
		t.Errorf("WardByCode(0040) = %+v, %v", w, ok)
	}

	if got := len(d.WardsInCounty(47)); got != 85 {
		t.Errorf("Nairobi wards = %d, want 85", got)
	}
	if d.WardsInCounty(99) != nil {
		t.Error("WardsInCounty(99) should be nil")
	}

	lari := d.WardsInConstituency("lari")
	if len(lari) == 0 {
		t.Fatal("no wards for Lari")
	}
	for _, w := range lari {
		if w.CountyCode != 22 {
			t.Errorf("Lari ward %s in county %d, want 22 (Kiambu)", w.Name, w.CountyCode)
		}
	}
}

func TestNullVotersBecomeNil(t *testing.T) {
	w, ok := Default().WardByCode("01811810")
	if !ok {
		t.Fatal("ward 01811810 (Della) missing")
	}
	if w.RegisteredVoters2022 != nil {
		t.Errorf("voters = %d, want nil", *w.RegisteredVoters2022)
	}
}

func TestEveryWardHasACounty(t *testing.T) {
	for _, w := range Default().wards {
		if w.CountyCode < 1 || w.CountyCode > 47 {
			t.Errorf("ward %s has county code %d", w.Code, w.CountyCode)
		}
	}
}

func TestSearch(t *testing.T) {
	d := Default()
	tests := []struct {
		query    string
		wantName string
		wantKind Kind
	}{
		{"nyeri", "Nyeri County", KindCounty},
		{"Kamukunji", "Kamukunji", KindConstituency},
		{"eastleigh south", "EASTLEIGH SOUTH", KindWard},
		{"nakru", "Nakuru County", KindCounty}, // typo
		{"tambua", "TAMB​UA", KindWard},        // source name hides a zero-width space
		{"lari kirenga", "LARI/KIRENGA", KindWard},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := d.Search(tt.query, 5)
			if len(got) == 0 {
				t.Fatalf("Search(%q) returned nothing", tt.query)
			}
			if got[0].Name != tt.wantName || got[0].Kind != tt.wantKind {
				t.Errorf("Search(%q)[0] = %s %q, want %s %q",
					tt.query, got[0].Kind, got[0].Name, tt.wantKind, tt.wantName)
			}
		})
	}

	if got := d.Search("", 5); got != nil {
		t.Errorf("empty query returned %d results", len(got))
	}
	if got := d.Search("zzzzqqq", 5); len(got) != 0 {
		t.Errorf("nonsense query returned %v", got)
	}
	if got := d.Search("a", 3); len(got) != 3 {
		t.Errorf("limit not applied: got %d results", len(got))
	}
}

func TestNormalize(t *testing.T) {
	tests := map[string]string{
		"Murang’a County":  "muranga",
		"Taita–Taveta":     "taita taveta",
		"LARI/KIRENGA":     "lari kirenga",
		"TAMB​UA":          "tambua",
		" Homa  Bay ":      "homa bay",
		"AGENG'A NANGUBA":  "agenga nanguba",
		"Kerugoya / Kutus": "kerugoya kutus",
	}
	for in, want := range tests {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadRejectsUnknownCounty(t *testing.T) {
	counties := `[{"code":1,"name":"Mombasa County"}]`
	wards := `[{"ward_code":"0001","name":"X","constituency_name":"Y","county_name":"Nowhere County"}]`
	_, err := Load(strings.NewReader(counties), strings.NewReader(wards))
	if err == nil || !strings.Contains(err.Error(), "Nowhere") {
		t.Errorf("err = %v, want unknown county error", err)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"kitten", "sitting", 3},
		{"nakuru", "nakru", 1},
		{"murang’a", "muranga", 1}, // one rune, not three bytes
	}
	for _, tt := range tests {
		if got := levenshtein([]rune(tt.a), []rune(tt.b)); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// Benchmarks run with: go test -bench=. -benchmem
func BenchmarkCountyByName(b *testing.B) {
	d := Default()
	for b.Loop() {
		d.CountyByName("Murang'a")
	}
}

func BenchmarkSearch(b *testing.B) {
	d := Default()
	for b.Loop() {
		d.Search("kibra", 10)
	}
}

// Example functions are compiled, run by go test (the output is checked
// against the "Output:" comment), and shown in the package's documentation.
func ExampleData_CountyByName() {
	c, ok := Default().CountyByName("taita taveta")
	fmt.Println(c.Code, c.Name, c.Headquarters, ok)
	// Output: 6 Taita–Taveta County Mwatate true
}

func ExampleData_Search() {
	for _, m := range Default().Search("makadara", 3) {
		fmt.Printf("%s %s\n", m.Kind, m.Name)
	}
	// Output:
	// constituency Makadara
	// ward MJI WA KALE/MAKADARA
}
