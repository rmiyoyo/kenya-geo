package kenyageo

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

//go:embed data/counties.json
var countiesJSON []byte

//go:embed data/wards.json
var wardsJSON []byte

//go:embed data/postoffices.json
var postOfficesJSON []byte

//go:embed data/wardshapes.json
var wardShapesJSON []byte

//go:embed data/countyshapes.json
var countyShapesJSON []byte

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type County struct {
	Code           int    `json:"code"`
	Name           string `json:"name"`
	ISOCode        string `json:"iso_code"`
	FormerProvince string `json:"former_province"`
	Headquarters   string `json:"headquarters"`

	Population         int     `json:"population"`
	PopulationMale     int     `json:"population_male"`
	PopulationFemale   int     `json:"population_female"`
	PopulationIntersex int     `json:"population_intersex"`
	AreaKm2            float64 `json:"area_km2"`

	Centroid   LatLng     `json:"centroid"`
	BBox       [4]float64 `json:"bbox"`
	Neighbours []int      `json:"neighbours"`
}

func (c County) Density() float64 {
	if c.AreaKm2 == 0 {
		return 0
	}
	return float64(c.Population) / c.AreaKm2
}

type Ward struct {
	Code         string `json:"ward_code"`
	Name         string `json:"name"`
	Constituency string `json:"constituency_name"`
	CountyCode   int    `json:"county_code"`
	CountyName   string `json:"county_name"`

	RegisteredVoters2022 *int `json:"registered_voters_2022"`

	Centroid *LatLng `json:"centroid"`
}

type PostOffice struct {
	Code         string  `json:"postal_code"`
	Name         string  `json:"name"`
	CountyCode   int     `json:"county_code"`
	CountySource string  `json:"county_source"`
	Location     *LatLng `json:"location"`
}

type Data struct {
	counties    []County
	wards       []Ward
	postOffices []PostOffice

	countyByCode   map[int]int
	countyByName   map[string]int
	wardByCode     map[string]int
	wardsByCounty  map[int][]int
	wardsByConst   map[string][]int
	constsByCounty map[int][]string
	constCounty    map[string]int

	postByCode   map[string][]int
	postByCounty map[int][]int
	postByName   map[string][]int

	wardShapes   func() []area
	countyShapes func() []area

	searchIndex []searchEntry
}

var defaultData = sync.OnceValue(func() *Data {
	d, err := Load(bytes.NewReader(countiesJSON), bytes.NewReader(wardsJSON), bytes.NewReader(postOfficesJSON))
	if err != nil {
		panic("kenyageo: embedded data is invalid: " + err.Error())
	}
	d.wardShapes = sync.OnceValue(func() []area {
		shapes, err := d.readWardShapes(bytes.NewReader(wardShapesJSON))
		if err != nil {
			panic("kenyageo: embedded ward shapes are invalid: " + err.Error())
		}
		return shapes
	})
	d.countyShapes = sync.OnceValue(func() []area {
		shapes, err := d.readCountyShapes(bytes.NewReader(countyShapesJSON))
		if err != nil {
			panic("kenyageo: embedded county shapes are invalid: " + err.Error())
		}
		return shapes
	})
	return d
})

func Default() *Data { return defaultData() }

func Load(counties, wards, postOffices io.Reader) (*Data, error) {
	d := &Data{}
	if err := json.NewDecoder(counties).Decode(&d.counties); err != nil {
		return nil, fmt.Errorf("decoding counties: %w", err)
	}
	if err := json.NewDecoder(wards).Decode(&d.wards); err != nil {
		return nil, fmt.Errorf("decoding wards: %w", err)
	}
	if err := json.NewDecoder(postOffices).Decode(&d.postOffices); err != nil {
		return nil, fmt.Errorf("decoding post offices: %w", err)
	}

	sort.Slice(d.counties, func(i, j int) bool { return d.counties[i].Code < d.counties[j].Code })
	sort.Slice(d.wards, func(i, j int) bool { return d.wards[i].Code < d.wards[j].Code })

	d.countyByCode = make(map[int]int, len(d.counties))
	d.countyByName = make(map[string]int, len(d.counties))
	for i, c := range d.counties {
		if _, dup := d.countyByCode[c.Code]; dup {
			return nil, fmt.Errorf("duplicate county code %d", c.Code)
		}
		d.countyByCode[c.Code] = i
		d.countyByName[normalize(c.Name)] = i
	}

	d.wardByCode = make(map[string]int, len(d.wards))
	d.wardsByCounty = make(map[int][]int)
	d.wardsByConst = make(map[string][]int)
	d.constsByCounty = make(map[int][]string)
	d.constCounty = make(map[string]int)

	for i := range d.wards {
		w := &d.wards[i]
		ci, ok := d.countyByCode[w.CountyCode]
		if !ok || normalize(d.counties[ci].Name) != normalize(w.CountyName) {
			return nil, fmt.Errorf("ward %s (%s): county %d %q does not match the counties data",
				w.Code, w.Name, w.CountyCode, w.CountyName)
		}
		if _, dup := d.wardByCode[w.Code]; dup {
			return nil, fmt.Errorf("duplicate ward code %s", w.Code)
		}
		d.wardByCode[w.Code] = i
		d.wardsByCounty[w.CountyCode] = append(d.wardsByCounty[w.CountyCode], i)

		key := normalize(w.Constituency)
		if _, seen := d.wardsByConst[key]; !seen {
			d.constsByCounty[w.CountyCode] = append(d.constsByCounty[w.CountyCode], w.Constituency)
			d.constCounty[key] = w.CountyCode
		}
		d.wardsByConst[key] = append(d.wardsByConst[key], i)
	}
	for _, names := range d.constsByCounty {
		sort.Strings(names)
	}

	sort.SliceStable(d.postOffices, func(i, j int) bool { return d.postOffices[i].Code < d.postOffices[j].Code })
	d.postByCode = make(map[string][]int, len(d.postOffices))
	d.postByCounty = make(map[int][]int)
	d.postByName = make(map[string][]int, len(d.postOffices))
	for i, p := range d.postOffices {
		if _, ok := d.countyByCode[p.CountyCode]; !ok && p.CountyCode != 0 {
			return nil, fmt.Errorf("post office %s (%s): unknown county %d", p.Code, p.Name, p.CountyCode)
		}
		d.postByCode[p.Code] = append(d.postByCode[p.Code], i)
		d.postByName[officeKey(p.Name)] = append(d.postByName[officeKey(p.Name)], i)
		if p.CountyCode != 0 {
			d.postByCounty[p.CountyCode] = append(d.postByCounty[p.CountyCode], i)
		}
	}
	d.buildSearchIndex()
	return d, nil
}

func (d *Data) Counties() []County {
	return append([]County(nil), d.counties...)
}

func (d *Data) CountyByCode(code int) (County, bool) {
	i, ok := d.countyByCode[code]
	if !ok {
		return County{}, false
	}
	return d.counties[i], true
}

func (d *Data) CountyByName(name string) (County, bool) {
	i, ok := d.countyByName[normalize(name)]
	if !ok {
		return County{}, false
	}
	return d.counties[i], true
}

func (d *Data) NeighbouringCounties(code int) []County {
	i, ok := d.countyByCode[code]
	if !ok {
		return nil
	}
	out := make([]County, 0, len(d.counties[i].Neighbours))
	for _, n := range d.counties[i].Neighbours {
		if j, ok := d.countyByCode[n]; ok {
			out = append(out, d.counties[j])
		}
	}
	return out
}

func (d *Data) WardByCode(code string) (Ward, bool) {
	i, ok := d.wardByCode[code]
	if !ok {
		return Ward{}, false
	}
	return d.wards[i], true
}

func (d *Data) WardsInCounty(countyCode int) []Ward {
	return d.pick(d.wardsByCounty[countyCode])
}

func (d *Data) Constituencies(countyCode int) []string {
	return append([]string(nil), d.constsByCounty[countyCode]...)
}

func (d *Data) WardsInConstituency(name string) []Ward {
	return d.pick(d.wardsByConst[normalize(name)])
}

func (d *Data) pick(idx []int) []Ward {
	if len(idx) == 0 {
		return nil
	}
	out := make([]Ward, len(idx))
	for i, j := range idx {
		out[i] = d.wards[j]
	}
	return out
}

func (d *Data) PostOffices(postalCode string) []PostOffice {
	return d.pickPost(d.postByCode[strings.TrimSpace(postalCode)])
}

func (d *Data) PostOfficesInCounty(countyCode int) []PostOffice {
	return d.pickPost(d.postByCounty[countyCode])
}

func (d *Data) pickPost(idx []int) []PostOffice {
	if len(idx) == 0 {
		return nil
	}
	out := make([]PostOffice, len(idx))
	for i, j := range idx {
		out[i] = d.postOffices[j]
	}
	return out
}
