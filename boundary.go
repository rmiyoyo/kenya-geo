package kenyageo

type Feature[P any] struct {
	Type       string   `json:"type"`
	Geometry   Geometry `json:"geometry"`
	Properties P        `json:"properties"`
}

type Geometry struct {
	Type        string           `json:"type"`
	Coordinates [][][][2]float64 `json:"coordinates"`
}

func (d *Data) WardBoundary(code string) (Feature[Ward], bool) {
	i, ok := d.wardByCode[code]
	if !ok || d.wardShapes == nil {
		return Feature[Ward]{}, false
	}
	g, ok := geometryOf(d.wardShapes(), i)
	if !ok {
		return Feature[Ward]{}, false
	}
	return Feature[Ward]{Type: "Feature", Geometry: g, Properties: d.wards[i]}, true
}

func (d *Data) CountyBoundary(code int) (Feature[County], bool) {
	i, ok := d.countyByCode[code]
	if !ok || d.countyShapes == nil {
		return Feature[County]{}, false
	}
	g, ok := geometryOf(d.countyShapes(), i)
	if !ok {
		return Feature[County]{}, false
	}
	return Feature[County]{Type: "Feature", Geometry: g, Properties: d.counties[i]}, true
}

func geometryOf(areas []area, index int) (Geometry, bool) {
	for _, a := range areas {
		if a.index != index {
			continue
		}
		coords := make([][][][2]float64, len(a.polygons))
		for i, p := range a.polygons {
			coords[i] = make([][][2]float64, len(p))
			for j, r := range p {
				coords[i][j] = closedRing(r, j == 0)
			}
		}
		return Geometry{Type: "MultiPolygon", Coordinates: coords}, true
	}
	return Geometry{}, false
}

func closedRing(r [][2]float64, outer bool) [][2]float64 {
	out := make([][2]float64, 0, len(r)+1)
	out = append(out, r...)
	if len(out) > 0 && out[0] != out[len(out)-1] {
		out = append(out, out[0])
	}
	if (signedArea(out) > 0) != outer {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out
}

func signedArea(r [][2]float64) float64 {
	s := 0.0
	for i := 0; i+1 < len(r); i++ {
		s += r[i][0]*r[i+1][1] - r[i+1][0]*r[i][1]
	}
	return s / 2
}
