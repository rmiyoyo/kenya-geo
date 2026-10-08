package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

type (
	ring    [][2]float64
	polygon []ring
)

type shape struct {
	Name     string
	ISO      string
	polygons []polygon
	Centroid LatLng
	BBox     [4]float64
	inside   [2]float64
}

func readShapes(path string) ([]shape, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fc struct {
		Features []struct {
			Properties struct {
				Name string `json:"shapeName"`
				ISO  string `json:"shapeISO"`
			} `json:"properties"`
			Geometry struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(b, &fc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	out := make([]shape, 0, len(fc.Features))
	for _, f := range fc.Features {
		var polys []polygon
		switch f.Geometry.Type {
		case "Polygon":
			var p polygon
			err = json.Unmarshal(f.Geometry.Coordinates, &p)
			polys = []polygon{p}
		case "MultiPolygon":
			err = json.Unmarshal(f.Geometry.Coordinates, &polys)
		default:
			err = fmt.Errorf("unsupported geometry %q", f.Geometry.Type)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, f.Properties.Name, err)
		}
		s := shape{Name: f.Properties.Name, ISO: f.Properties.ISO, polygons: polys}
		s.Centroid = s.centroid()
		s.BBox = s.bbox()
		s.inside = s.interiorPoint()
		out = append(out, s)
	}
	return out, nil
}

func (s shape) centroid() LatLng {
	var area, cx, cy float64
	for _, p := range s.polygons {
		for i, r := range p {
			a, x, y := r.areaCentroid()
			if i > 0 {
				a = -math.Abs(a)
			} else {
				a = math.Abs(a)
			}
			area += a
			cx += x * a
			cy += y * a
		}
	}
	return LatLng{Lat: round5(cy / area), Lng: round5(cx / area)}
}

func (r ring) areaCentroid() (area, cx, cy float64) {
	var a, x, y float64
	for i := range r {
		p, q := r[i], r[(i+1)%len(r)]
		f := p[0]*q[1] - q[0]*p[1]
		a += f
		x += (p[0] + q[0]) * f
		y += (p[1] + q[1]) * f
	}
	if a == 0 {
		return 0, 0, 0
	}
	return a / 2, x / (3 * a), y / (3 * a)
}

func (s shape) bbox() [4]float64 {
	b := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, p := range s.polygons {
		for _, pt := range p[0] {
			b[0], b[1] = min(b[0], pt[0]), min(b[1], pt[1])
			b[2], b[3] = max(b[2], pt[0]), max(b[3], pt[1])
		}
	}
	for i := range b {
		b[i] = round5(b[i])
	}
	return b
}

func (s shape) contains(pt [2]float64) bool {
	for _, p := range s.polygons {
		if p[0].contains(pt) {
			inHole := false
			for _, h := range p[1:] {
				if h.contains(pt) {
					inHole = true
					break
				}
			}
			if !inHole {
				return true
			}
		}
	}
	return false
}

func (r ring) contains(pt [2]float64) bool {
	in := false
	for i := range r {
		p, q := r[i], r[(i+1)%len(r)]
		if (p[1] > pt[1]) != (q[1] > pt[1]) &&
			pt[0] < (q[0]-p[0])*(pt[1]-p[1])/(q[1]-p[1])+p[0] {
			in = !in
		}
	}
	return in
}

func (s shape) interiorPoint() [2]float64 {
	c := [2]float64{s.Centroid.Lng, s.Centroid.Lat}
	if s.contains(c) {
		return c
	}
	var xs []float64
	for _, p := range s.polygons {
		for _, r := range p {
			for i := range r {
				a, b := r[i], r[(i+1)%len(r)]
				if (a[1] > c[1]) != (b[1] > c[1]) {
					xs = append(xs, (b[0]-a[0])*(c[1]-a[1])/(b[1]-a[1])+a[0])
				}
			}
		}
	}
	sort.Float64s(xs)
	best := c
	widest := -1.0
	for i := 0; i+1 < len(xs); i += 2 {
		if w := xs[i+1] - xs[i]; w > widest {
			widest = w
			best = [2]float64{(xs[i] + xs[i+1]) / 2, c[1]}
		}
	}
	return best
}

func round5(f float64) float64 { return math.Round(f*1e5) / 1e5 }
