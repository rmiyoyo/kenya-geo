package kenyageo

import (
	"fmt"
	"strings"
	"testing"
)

func TestCountyAt(t *testing.T) {
	tests := []struct {
		place    string
		lat, lng float64
		county   int
	}{
		{"KICC", -1.2884, 36.8233, 47},
		{"Fort Jesus", -4.0626, 39.6795, 1},
		{"Lodwar", 3.1191, 35.5973, 23},
		{"Garissa", -0.4532, 39.6461, 7},
		{"Kisumu", -0.0917, 34.768, 42},
		{"Eldoret", 0.5143, 35.2698, 27},
		{"Mandera", 3.9366, 41.8670, 9},
	}
	d := Default()
	for _, tt := range tests {
		c, ok := d.CountyAt(tt.lat, tt.lng)
		if !ok || c.Code != tt.county {
			t.Errorf("%s: CountyAt = %d %s, %v, want %d", tt.place, c.Code, c.Name, ok, tt.county)
		}
	}
	for _, p := range [][2]float64{{-4.5, 40.5}, {9.03, 38.74}} {
		if c, ok := d.CountyAt(p[0], p[1]); ok {
			t.Errorf("CountyAt(%v, %v) = %s, want nothing", p[0], p[1], c.Name)
		}
	}
}

func TestCountyAtAgreesWithWards(t *testing.T) {
	d := Default()
	checked, disagree := 0, 0
	for _, w := range d.wards {
		if w.Centroid == nil {
			continue
		}
		c, ok := d.CountyAt(w.Centroid.Lat, w.Centroid.Lng)
		if !ok {
			continue
		}
		checked++
		if c.Code != w.CountyCode {
			disagree++
			t.Logf("ward %s %s (%s): centroid is in %s", w.Code, w.Name, w.CountyName, c.Name)
		}
	}
	if checked < 1400 || disagree > 12 {
		t.Errorf("checked %d ward centroids, %d in another county; want at least 1400 and at most 12", checked, disagree)
	}
}

func TestCountyShapes(t *testing.T) {
	d := Default()
	seen := map[int]bool{}
	for _, s := range d.countyShapes() {
		seen[d.counties[s.index].Code] = true
	}
	if len(seen) != 47 {
		t.Errorf("%d counties have a boundary, want 47", len(seen))
	}

	bad := `[{"code":99,"polygons":[[[[0,0],[1,0],[1,1]]]]}]`
	if err := d.LoadCountyShapes(strings.NewReader(bad)); err == nil {
		t.Error("loading a shape for county 99 succeeded")
	}
}

func BenchmarkCountyAt(b *testing.B) {
	d := Default()
	d.CountyAt(0, 0)
	for b.Loop() {
		d.CountyAt(-1.2884, 36.8233)
	}
}

func ExampleData_CountyAt() {
	c, ok := Default().CountyAt(-0.0917, 34.768)
	fmt.Println(c.Code, c.Name, ok)
	// Output: 42 Kisumu County true
}
