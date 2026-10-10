package kenyageo

import (
	"cmp"
	"math"
	"slices"
)

type NearbyPostOffice struct {
	PostOffice
	DistanceKm float64 `json:"distance_km"`
}

func (d *Data) PostOfficesNear(lat, lng float64, n int) []NearbyPostOffice {
	if n <= 0 {
		return nil
	}
	type candidate struct {
		i  int
		km float64
	}
	cands := make([]candidate, 0, len(d.postOffices))
	for i, p := range d.postOffices {
		if p.Location != nil {
			cands = append(cands, candidate{i, distanceKm(lat, lng, p.Location.Lat, p.Location.Lng)})
		}
	}
	slices.SortFunc(cands, func(a, b candidate) int {
		if c := cmp.Compare(a.km, b.km); c != 0 {
			return c
		}
		return cmp.Compare(d.postOffices[a.i].Code, d.postOffices[b.i].Code)
	})
	out := make([]NearbyPostOffice, min(n, len(cands)))
	for k := range out {
		out[k] = NearbyPostOffice{PostOffice: d.postOffices[cands[k].i], DistanceKm: cands[k].km}
	}
	return out
}

func distanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0088
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}
