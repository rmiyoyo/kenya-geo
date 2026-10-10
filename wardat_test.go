package kenyageo

import (
	"fmt"
	"strings"
	"testing"
)

func TestWardAt(t *testing.T) {
	tests := []struct {
		place        string
		lat, lng     float64
		ward         string
		constituency string
		county       int
	}{
		{"KICC", -1.2884, 36.8233, "1439", "Starehe", 47},
		{"JKIA", -1.3197, 36.9257, "1423", "Embakasi East", 47},
		{"Fort Jesus", -4.0626, 39.6795, "0026", "Mvita", 1},
		{"Lodwar", 3.1191, 35.5973, "0627", "Turkana Central", 23},
		{"Nakuru town", -0.2944, 36.0778, "0876", "Nakuru Town East", 32},
		{"Kitale", 1.0157, 35.0062, "", "", 26},
	}
	d := Default()
	for _, tt := range tests {
		t.Run(tt.place, func(t *testing.T) {
			w, ok := d.WardAt(tt.lat, tt.lng)
			if !ok {
				t.Fatalf("WardAt(%v, %v) found nothing", tt.lat, tt.lng)
			}
			if w.CountyCode != tt.county {
				t.Errorf("county = %d (%s), want %d", w.CountyCode, w.CountyName, tt.county)
			}
			if tt.ward != "" && w.Code != tt.ward {
				t.Errorf("ward = %s %s, want %s", w.Code, w.Name, tt.ward)
			}
			if tt.constituency != "" && w.Constituency != tt.constituency {
				t.Errorf("constituency = %s, want %s", w.Constituency, tt.constituency)
			}
		})
	}
}

func TestWardAtOutsideKenya(t *testing.T) {
	for _, p := range [][2]float64{{-4.5, 40.5}, {9.03, 38.74}, {0, 0}} {
		if w, ok := Default().WardAt(p[0], p[1]); ok {
			t.Errorf("WardAt(%v, %v) = %s %s, want nothing", p[0], p[1], w.Code, w.Name)
		}
	}
}

func TestWardAtFindsWardCentroids(t *testing.T) {
	d := Default()
	same, otherCounty := 0, 0
	for _, w := range d.wards {
		if w.Centroid == nil {
			continue
		}
		got, ok := d.WardAt(w.Centroid.Lat, w.Centroid.Lng)
		if ok && got.Code == w.Code {
			same++
		}
		if ok && got.CountyCode != w.CountyCode {
			otherCounty++
			t.Logf("ward %s %s: centroid lands in %s %s, %s", w.Code, w.Name, got.Code, got.Name, got.CountyName)
		}
	}
	if otherCounty > 5 {
		t.Errorf("%d ward centroids land in another county, want at most 5", otherCounty)
	}
	if same < 1400 {
		t.Errorf("%d ward centroids land in their own ward, want at least 1400", same)
	}
}

func TestWardShapes(t *testing.T) {
	d := Default()
	shapes := d.wardShapes()
	if len(shapes) < 1430 {
		t.Errorf("%d ward shapes, want at least 1430", len(shapes))
	}
	seen := map[int]bool{}
	for _, s := range shapes {
		if seen[s.index] {
			t.Errorf("ward %s has two shapes", d.wards[s.index].Code)
		}
		seen[s.index] = true
	}
}

func TestWardAtWithoutShapes(t *testing.T) {
	counties := `[{"code":47,"name":"Nairobi County"}]`
	wards := `[{"ward_code":"1439","name":"NAIROBI CENTRAL","constituency_name":"Starehe","county_code":47,"county_name":"Nairobi County"}]`
	d, err := Load(strings.NewReader(counties), strings.NewReader(wards), strings.NewReader("[]"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.WardAt(-1.2884, 36.8233); ok {
		t.Error("WardAt found a ward before any shapes were loaded")
	}

	square := `[{"ward_code":"1439","polygons":[[[[36.8,-1.3],[36.9,-1.3],[36.9,-1.2],[36.8,-1.2]]]]}]`
	if err := d.LoadWardShapes(strings.NewReader(square)); err != nil {
		t.Fatal(err)
	}
	if w, ok := d.WardAt(-1.2884, 36.8233); !ok || w.Code != "1439" {
		t.Errorf("WardAt = %s, %v, want 1439", w.Code, ok)
	}

	unknown := `[{"ward_code":"9999","polygons":[[[[0,0],[1,0],[1,1]]]]}]`
	if err := d.LoadWardShapes(strings.NewReader(unknown)); err == nil || !strings.Contains(err.Error(), "9999") {
		t.Errorf("err = %v, want unknown ward error", err)
	}
}

func TestRingContainsHole(t *testing.T) {
	s := area{polygons: [][][][2]float64{{
		{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
		{{4, 4}, {6, 4}, {6, 6}, {4, 6}},
	}}}
	tests := map[[2]float64]bool{{1, 1}: true, {5, 5}: false, {11, 5}: false, {9, 9}: true}
	for pt, want := range tests {
		if got := s.contains(pt); got != want {
			t.Errorf("contains(%v) = %v, want %v", pt, got, want)
		}
	}
}

func BenchmarkWardAt(b *testing.B) {
	d := Default()
	d.WardAt(0, 0)
	for b.Loop() {
		d.WardAt(-1.2884, 36.8233)
	}
}

func ExampleData_WardAt() {
	w, ok := Default().WardAt(-1.2884, 36.8233)
	fmt.Println(w.Code, w.Name, w.Constituency, w.CountyName, ok)
	// Output: 1439 NAIROBI CENTRAL Starehe Nairobi County true
}
