package kenyageo

import (
	"encoding/json"
	"fmt"
	"io"
)

func (d *Data) LoadWardShapes(r io.Reader) error {
	shapes, err := d.readWardShapes(r)
	if err != nil {
		return err
	}
	d.wardShapes = func() []area { return shapes }
	return nil
}

func (d *Data) readWardShapes(r io.Reader) ([]area, error) {
	var raw []struct {
		Code     string           `json:"ward_code"`
		Polygons [][][][2]float64 `json:"polygons"`
	}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding ward shapes: %w", err)
	}
	shapes := make([]area, 0, len(raw))
	for _, s := range raw {
		i, ok := d.wardByCode[s.Code]
		if !ok {
			return nil, fmt.Errorf("ward shape for unknown ward %s", s.Code)
		}
		a, err := newArea(i, s.Polygons)
		if err != nil {
			return nil, fmt.Errorf("ward shape %s: %w", s.Code, err)
		}
		shapes = append(shapes, a)
	}
	return shapes, nil
}

func (d *Data) WardAt(lat, lng float64) (Ward, bool) {
	if d.wardShapes == nil {
		return Ward{}, false
	}
	i, ok := findArea(d.wardShapes(), lat, lng)
	if !ok {
		return Ward{}, false
	}
	return d.wards[i], true
}
