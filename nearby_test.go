package kenyageo

import (
	"fmt"
	"math"
	"testing"
)

func TestDistanceKm(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lng1, lat2, lng2 float64
		want                   float64
	}{
		{"same point", -1.2864, 36.8172, -1.2864, 36.8172, 0},
		{"one degree of latitude", 0, 37, 1, 37, 111.2},
		{"Nairobi to Mombasa", -1.2864, 36.8172, -4.0435, 39.6682, 440.7},
	}
	for _, tt := range tests {
		if got := distanceKm(tt.lat1, tt.lng1, tt.lat2, tt.lng2); math.Abs(got-tt.want) > 0.1 {
			t.Errorf("%s: %.2f km, want %.1f", tt.name, got, tt.want)
		}
	}
}

func TestPostOfficesNear(t *testing.T) {
	d := Default()
	got := d.PostOfficesNear(3.1191, 35.5973, 3)
	if len(got) != 3 || got[0].Code != "30500" || got[0].DistanceKm > 1 {
		t.Fatalf("near Lodwar = %+v, want Lodwar 30500 first", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i].DistanceKm < got[i-1].DistanceKm {
			t.Errorf("results not sorted by distance: %v", got)
		}
	}
	for _, p := range d.PostOfficesNear(-0.2833, 36.0667, 10) {
		if p.Location == nil {
			t.Errorf("%s %s has no location", p.Code, p.Name)
		}
	}
	if n := len(d.PostOfficesNear(0, 37, 10000)); n != 553 {
		t.Errorf("asking for everything returned %d offices, want the 553 with a location", n)
	}
	if got := d.PostOfficesNear(0, 37, 0); got != nil {
		t.Errorf("n=0 returned %v", got)
	}
}

func BenchmarkPostOfficesNear(b *testing.B) {
	d := Default()
	for b.Loop() {
		d.PostOfficesNear(-1.2884, 36.8233, 5)
	}
}

func ExampleData_PostOfficesNear() {
	for _, p := range Default().PostOfficesNear(-0.2833, 36.0667, 2) {
		fmt.Printf("%s %s %.1f km\n", p.Code, p.Name, p.DistanceKm)
	}
	// Output:
	// 20100 Nakuru 2.7 km
	// 20112 Lanet 8.2 km
}
