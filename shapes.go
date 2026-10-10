package kenyageo

import (
	"errors"
	"math"
)

type area struct {
	index    int
	bbox     [4]float64
	polygons [][][][2]float64
}

func newArea(index int, polygons [][][][2]float64) (area, error) {
	b := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, p := range polygons {
		if len(p) == 0 {
			return area{}, errors.New("empty polygon")
		}
		for _, pt := range p[0] {
			b[0], b[1] = min(b[0], pt[0]), min(b[1], pt[1])
			b[2], b[3] = max(b[2], pt[0]), max(b[3], pt[1])
		}
	}
	return area{index: index, bbox: b, polygons: polygons}, nil
}

func findArea(areas []area, lat, lng float64) (int, bool) {
	pt := [2]float64{lng, lat}
	for _, a := range areas {
		if lng < a.bbox[0] || lng > a.bbox[2] || lat < a.bbox[1] || lat > a.bbox[3] {
			continue
		}
		if a.contains(pt) {
			return a.index, true
		}
	}
	return 0, false
}

func (a area) contains(pt [2]float64) bool {
	for _, p := range a.polygons {
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
