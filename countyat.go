package kenyageo

import (
	"encoding/json"
	"fmt"
	"io"
)

func (d *Data) LoadCountyShapes(r io.Reader) error {
	shapes, err := d.readCountyShapes(r)
	if err != nil {
		return err
	}
	d.countyShapes = func() []area { return shapes }
	return nil
}

func (d *Data) readCountyShapes(r io.Reader) ([]area, error) {
	var raw []struct {
		Code     int              `json:"code"`
		Polygons [][][][2]float64 `json:"polygons"`
	}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding county shapes: %w", err)
	}
	shapes := make([]area, 0, len(raw))
	for _, s := range raw {
		i, ok := d.countyByCode[s.Code]
		if !ok {
			return nil, fmt.Errorf("county shape for unknown county %d", s.Code)
		}
		a, err := newArea(i, s.Polygons)
		if err != nil {
			return nil, fmt.Errorf("county shape %d: %w", s.Code, err)
		}
		shapes = append(shapes, a)
	}
	return shapes, nil
}

func (d *Data) CountyAt(lat, lng float64) (County, bool) {
	if d.countyShapes == nil {
		return County{}, false
	}
	i, ok := findArea(d.countyShapes(), lat, lng)
	if !ok {
		return County{}, false
	}
	return d.counties[i], true
}
