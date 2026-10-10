package kenyageo

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
)

type wardShape struct {
	ward     int
	bbox     [4]float64
	polygons [][][][2]float64
}

func (d *Data) LoadWardShapes(r io.Reader) error {
	shapes, err := d.readWardShapes(r)
	if err != nil {
		return err
	}
	d.shapes = func() []wardShape { return shapes }
	return nil
}

func (d *Data) readWardShapes(r io.Reader) ([]wardShape, error) {
	var raw []struct {
		Code     string           `json:"ward_code"`
		Polygons [][][][2]float64 `json:"polygons"`
	}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding ward shapes: %w", err)
	}
	shapes := make([]wardShape, 0, len(raw))
	for _, s := range raw {
		i, ok := d.wardByCode[s.Code]
		if !ok {
			return nil, fmt.Errorf("ward shape for unknown ward %s", s.Code)
		}
		b := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
		for _, p := range s.Polygons {
			if len(p) == 0 {
				return nil, fmt.Errorf("ward shape %s: empty polygon", s.Code)
			}
			for _, pt := range p[0] {
				b[0], b[1] = min(b[0], pt[0]), min(b[1], pt[1])
				b[2], b[3] = max(b[2], pt[0]), max(b[3], pt[1])
			}
		}
		shapes = append(shapes, wardShape{ward: i, bbox: b, polygons: s.Polygons})
	}
	return shapes, nil
}

func (d *Data) WardAt(lat, lng float64) (Ward, bool) {
	if d.shapes == nil {
		return Ward{}, false
	}
	pt := [2]float64{lng, lat}
	for _, s := range d.shapes() {
		if lng < s.bbox[0] || lng > s.bbox[2] || lat < s.bbox[1] || lat > s.bbox[3] {
			continue
		}
		if s.contains(pt) {
			return d.wards[s.ward], true
		}
	}
	return Ward{}, false
}

func (s wardShape) contains(pt [2]float64) bool {
	for _, p := range s.polygons {
		if !ringContains(p[0], pt) {
			continue
		}
		inHole := false
		for _, h := range p[1:] {
			if ringContains(h, pt) {
				inHole = true
				break
			}
		}
		if !inHole {
			return true
		}
	}
	return false
}

func ringContains(r [][2]float64, pt [2]float64) bool {
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
