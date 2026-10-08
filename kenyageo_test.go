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
	if constituencies != 290 {
		t.Errorf("constituencies = %d, want 290", constituencies)
	}
}

func TestCensusTotals(t *testing.T) {
	var pop, male, female, intersex int
	for _, c := range Default().Counties() {
		if got := c.PopulationMale + c.PopulationFemale + c.PopulationIntersex; got != c.Population {
			t.Errorf("%s: male+female+intersex = %d, want %d", c.Name, got, c.Population)
		}
		pop += c.Population
		male += c.PopulationMale
		female += c.PopulationFemale
		intersex += c.PopulationIntersex
	}
	if pop != 47_564_296 || male != 23_548_056 || female != 24_014_716 || intersex != 1_524 {
		t.Errorf("national totals = %d (m %d, f %d, i %d), want 47564296 (m 23548056, f 24014716, i 1524)",
			pop, male, female, intersex)
	}
}

func TestCountyFields(t *testing.T) {
	d := Default()
	provinces := map[string]int{}
	isos := map[string]bool{}
	for _, c := range d.Counties() {
		provinces[c.FormerProvince]++
		isos[c.ISOCode] = true
		if !strings.HasPrefix(c.ISOCode, "KE-") {
			t.Errorf("%s: ISO code %q", c.Name, c.ISOCode)
		}
		if !inKenya(c.Centroid) {
			t.Errorf("%s: centroid %v is outside Kenya", c.Name, c.Centroid)
		}
		if strings.ContainsAny(c.Name, "–’") {
			t.Errorf("%s: name has non-ASCII punctuation", c.Name)
		}
	}
	if len(provinces) != 8 || len(isos) != 47 {
		t.Errorf("former provinces = %v, distinct ISO codes = %d", provinces, len(isos))
	}

	kilifi, _ := d.CountyByCode(3)
	if kilifi.Population != 1_453_787 {
		t.Errorf("Kilifi population = %d, want 1453787", kilifi.Population)
	}
	turkana, _ := d.CountyByCode(23)
	if turkana.FormerProvince != "Rift Valley" || turkana.ISOCode != "KE-43" {
		t.Errorf("Turkana = %q %q", turkana.FormerProvince, turkana.ISOCode)
	}
	nairobi, _ := d.CountyByCode(47)
	if got := nairobi.Density(); got < 6000 || got > 6500 {
		t.Errorf("Nairobi density = %.0f, want about 6247", got)
	}
}

func TestWardCentroids(t *testing.T) {
	d := Default()
	missing := 0
	for _, w := range d.wards {
		if w.Centroid == nil {
			missing++
			continue
		}

		c, _ := d.CountyByCode(w.CountyCode)
		p := *w.Centroid
		const m = 0.1
		if p.Lng < c.BBox[0]-m || p.Lng > c.BBox[2]+m || p.Lat < c.BBox[1]-m || p.Lat > c.BBox[3]+m {
			t.Errorf("ward %s %s: centroid %v outside %s", w.Code, w.Name, p, c.Name)
		}
	}
	if missing > 30 {
		t.Errorf("%d wards have no centroid, want at most 30", missing)
	}
}

func TestPostOffices(t *testing.T) {
	d := Default()
	if got := len(d.postOffices); got != 890 {
		t.Errorf("post offices = %d, want 890", got)
	}

	gpo := d.PostOffices("00100")
	if len(gpo) != 1 || gpo[0].Name != "Nairobi Gpo" || gpo[0].CountyCode != 47 {
		t.Errorf("PostOffices(00100) = %+v", gpo)
	}
	if got := len(d.PostOffices("40629")); got != 2 {
		t.Errorf("PostOffices(40629) = %d offices, want 2 (Mudhiero and Ndere share it)", got)
	}
	if d.PostOffices("99999") != nil {
		t.Error("PostOffices(99999) should be nil")
	}

	singore := d.PostOffices("30703")
	if len(singore) != 1 || singore[0].CountyCode != 28 {
		t.Errorf("Singore = %+v, want Elgeyo-Marakwet (28)", singore)
	}

	located, unknownCounty := 0, 0
	for _, p := range d.postOffices {
		if p.Location != nil {
			located++
			if !inKenya(*p.Location) {
				t.Errorf("post office %s %s: location %v outside Kenya", p.Code, p.Name, *p.Location)
			}
		}
		if p.CountyCode == 0 {
			unknownCounty++
		}
	}
	if located != 553 || unknownCounty != 0 {
		t.Errorf("located = %d, unknown county = %d; want 553 and 0", located, unknownCounty)
	}

	for code, want := range map[string]int{"00202": 47, "50301": 38, "40639": 41, "50407": 40, "90216": 15} {
		got := d.PostOffices(code)
		if len(got) == 0 || got[0].CountyCode != want {
			t.Errorf("PostOffices(%s) = %+v, want county %d", code, got, want)
		}
	}

	total := 0
	for _, c := range d.Counties() {
		total += len(d.PostOfficesInCounty(c.Code))
	}
	if total != 890 {
		t.Errorf("post offices across counties = %d, want 890", total)
	}
}

func inKenya(p LatLng) bool {
	return p.Lat > -4.8 && p.Lat < 5.1 && p.Lng > 33.8 && p.Lng < 42
}

func TestCountyByName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
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

	gatunduNorth := d.WardsInConstituency("Gatundu North")
	if len(gatunduNorth) != 4 {
		t.Errorf("Gatundu North wards = %d, want 4", len(gatunduNorth))
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
	w, ok := Default().WardByCode("0181")
	if !ok || w.Name != "DELLA" {
		t.Fatalf("ward 0181 = %+v, want DELLA", w)
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
		{"nakru", "Nakuru County", KindCounty},
		{"tambua", "TAMBUA", KindWard},
		{"lari kirenga", "LARI/KIRENGA", KindWard},
		{"kenyatta hospital", "Kenyatta Hospital", KindPostOffice},
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
		"TAMB\u200bUA":     "tambua",
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
	wards := `[{"ward_code":"0001","name":"X","constituency_name":"Y","county_code":1,"county_name":"Nowhere County"}]`
	_, err := Load(strings.NewReader(counties), strings.NewReader(wards), strings.NewReader("[]"))
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
		{"murang’a", "muranga", 1},
	}
	for _, tt := range tests {
		if got := levenshtein([]rune(tt.a), []rune(tt.b)); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

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

func ExampleData_CountyByName() {
	c, ok := Default().CountyByName("taita taveta")
	fmt.Println(c.Code, c.Name, c.Headquarters, ok)
	// Output: 6 Taita-Taveta County Mwatate true
}

func ExampleData_Search() {
	for _, m := range Default().Search("makadara", 3) {
		fmt.Printf("%s %s\n", m.Kind, m.Name)
	}
	// Output:
	// constituency Makadara
	// post office Makadara
	// ward MJI WA KALE/MAKADARA
}
