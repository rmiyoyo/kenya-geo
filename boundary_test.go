package kenyageo

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestWardBoundary(t *testing.T) {
	d := Default()
	f, ok := d.WardBoundary("1439")
	if !ok {
		t.Fatal("no boundary for ward 1439")
	}
	if f.Type != "Feature" || f.Geometry.Type != "MultiPolygon" || f.Properties.Name != "NAIROBI CENTRAL" {
		t.Errorf("feature = %s %s %s", f.Type, f.Geometry.Type, f.Properties.Name)
	}
	checkRings(t, f.Geometry)
	if _, ok := d.WardBoundary("9999"); ok {
		t.Error("found a boundary for ward 9999")
	}
}

func TestCountyBoundaries(t *testing.T) {
	d := Default()
	for _, c := range d.Counties() {
		f, ok := d.CountyBoundary(c.Code)
		if !ok {
			t.Errorf("no boundary for county %d %s", c.Code, c.Name)
			continue
		}
		checkRings(t, f.Geometry)
	}
	if _, ok := d.CountyBoundary(48); ok {
		t.Error("found a boundary for county 48")
	}
}

func TestBoundaryDoesNotShareShapes(t *testing.T) {
	d := Default()
	f, _ := d.WardBoundary("1439")
	f.Geometry.Coordinates[0][0][0] = [2]float64{0, 0}
	if w, ok := d.WardAt(-1.2884, 36.8233); !ok || w.Code != "1439" {
		t.Errorf("changing a returned boundary changed WardAt: %s %v", w.Code, ok)
	}
	g, _ := d.WardBoundary("1439")
	if g.Geometry.Coordinates[0][0][0] == [2]float64{0, 0} {
		t.Error("changing a returned boundary changed the next one")
	}
}

func TestClosedRing(t *testing.T) {
	cw := [][2]float64{{0, 0}, {0, 1}, {1, 1}, {1, 0}}
	outer := closedRing(cw, true)
	if len(outer) != 5 || outer[0] != outer[4] || signedArea(outer) <= 0 {
		t.Errorf("outer ring = %v", outer)
	}
	if hole := closedRing(outer, false); signedArea(hole) >= 0 {
		t.Errorf("hole = %v", hole)
	}
}

func checkRings(t *testing.T, g Geometry) {
	t.Helper()
	for _, p := range g.Coordinates {
		for j, r := range p {
			if len(r) < 4 || r[0] != r[len(r)-1] {
				t.Fatalf("ring %d is not closed: %d points", j, len(r))
			}
			if (signedArea(r) > 0) != (j == 0) {
				t.Fatalf("ring %d winds the wrong way", j)
			}
		}
	}
}

func ExampleData_CountyBoundary() {
	f, _ := Default().CountyBoundary(47)
	b, _ := json.Marshal(f)
	fmt.Println(f.Properties.Name, f.Geometry.Type, len(b) > 1000)
	// Output: Nairobi County MultiPolygon true
}
