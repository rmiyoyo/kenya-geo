// Package kenyageo holds Kenya's 47 counties, their constituencies and 1,450
// wards, and 890 post offices, with 2019 census figures, IEBC registered
// voters and boundaries, all embedded in the package.
//
// Start with [Default], which loads the embedded data once and is safe for
// concurrent use:
//
//	geo := kenyageo.Default()
//	nakuru, _ := geo.CountyByName("Nakuru")
//	ward, _ := geo.WardAt(-1.2884, 36.8233)
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

//go:embed data/kenya.json
var kenyaJSON []byte

//go:embed data/wardshapes.json
var wardShapesJSON []byte

//go:embed data/countyshapes.json
var countyShapesJSON []byte

// LatLng is a point in WGS 84 decimal degrees.
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// County is one of Kenya's 47 counties, with its 2019 census figures,
// location and neighbours.
type County struct {
	Code           int    `json:"code"`
	Name           string `json:"name"`
	ISOCode        string `json:"iso_code"`
	FormerProvince string `json:"former_province"`
	Headquarters   string `json:"headquarters"`
	// HeadquartersAliases lists other names in use for the headquarters
	// town, such as Vihiga for Mbale.
	HeadquartersAliases []string `json:"headquarters_aliases,omitempty"`

	Population         int     `json:"population"`
	PopulationMale     int     `json:"population_male"`
	PopulationFemale   int     `json:"population_female"`
	PopulationIntersex int     `json:"population_intersex"`
	PopulationUrban    int     `json:"population_urban"`
	PopulationRural    int     `json:"population_rural"`
	AreaKm2            float64 `json:"area_km2"`

	Centroid   LatLng     `json:"centroid"`
	BBox       [4]float64 `json:"bbox"`
	Neighbours []int      `json:"neighbours"`

	Living   Living   `json:"living"`
	Religion Religion `json:"religion"`
}

// Religion counts people in households by religious affiliation in the
// 2019 census. The groups add up to Total.
type Religion struct {
	Total             int `json:"total"`
	Catholic          int `json:"catholic"`
	Protestant        int `json:"protestant"`
	Evangelical       int `json:"evangelical"`
	AfricanInstituted int `json:"african_instituted"`
	Orthodox          int `json:"orthodox"`
	OtherChristian    int `json:"other_christian"`
	Islam             int `json:"islam"`
	Hindu             int `json:"hindu"`
	Traditionist      int `json:"traditionist"`
	OtherReligion     int `json:"other_religion"`
	NoReligion        int `json:"no_religion"`
	DontKnow          int `json:"dont_know"`
	NotStated         int `json:"not_stated"`
}

// Christian returns the number of people in all the Christian groups.
func (r Religion) Christian() int {
	return r.Catholic + r.Protestant + r.Evangelical + r.AfricanInstituted + r.Orthodox + r.OtherChristian
}

// Pct returns n as a percentage of Total, or 0 when Total is 0.
func (r Religion) Pct(n int) float64 {
	if r.Total == 0 {
		return 0
	}
	return 100 * float64(n) / float64(r.Total)
}

// National holds the 2019 census figures for Kenya as a whole.
type National struct {
	PopulationUrban int      `json:"population_urban"`
	PopulationRural int      `json:"population_rural"`
	Living          Living   `json:"living"`
	Religion        Religion `json:"religion"`
}

// Living holds living conditions from the 2019 census for a county or for
// Kenya. The fields ending in Pct are percentages from 0 to 100.
type Living struct {
	Households          int     `json:"households"`
	HouseholdSize       float64 `json:"average_household_size"`
	ElectricityPct      float64 `json:"electricity_pct"`
	PipedWaterPct       float64 `json:"piped_water_pct"`
	InternetPct         float64 `json:"internet_pct"`
	MobilePhonePct      float64 `json:"mobile_phone_pct"`
	SchoolAttendancePct float64 `json:"school_attendance_pct"`
}

// UrbanPct returns the percentage of the county's people who live in urban
// areas.
func (c County) UrbanPct() float64 {
	if c.Population == 0 {
		return 0
	}
	return 100 * float64(c.PopulationUrban) / float64(c.Population)
}

// Density returns people per square kilometre.
func (c County) Density() float64 {
	if c.AreaKm2 == 0 {
		return 0
	}
	return float64(c.Population) / c.AreaKm2
}

// Ward is an IEBC electoral ward, with its constituency, county and 2022
// registered voters.
type Ward struct {
	Code         string `json:"ward_code"`
	Name         string `json:"name"`
	Constituency string `json:"constituency_name"`
	CountyCode   int    `json:"county_code"`
	CountyName   string `json:"county_name"`

	RegisteredVoters2022 *int `json:"registered_voters_2022"`

	Centroid *LatLng `json:"centroid"`
}

// PostOffice is a post office and the postal code it serves.
type PostOffice struct {
	Code           string  `json:"postal_code"`
	Name           string  `json:"name"`
	CountyCode     int     `json:"county_code"`
	CountySource   string  `json:"county_source"`
	Location       *LatLng `json:"location"`
	LocationSource string  `json:"location_source"`
}

// Data holds the counties, wards and post offices and answers questions
// about them. Use [Default] for the embedded data, or [Load] for your own.
// A Data is safe for concurrent use once loaded.
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

	national *National
}

var defaultData = sync.OnceValue(func() *Data {
	d, err := Load(bytes.NewReader(countiesJSON), bytes.NewReader(wardsJSON), bytes.NewReader(postOfficesJSON))
	if err != nil {
		panic("kenyageo: embedded data is invalid: " + err.Error())
	}
	var kenya National
	if err := json.Unmarshal(kenyaJSON, &kenya); err != nil {
		panic("kenyageo: embedded national figures are invalid: " + err.Error())
	}
	d.national = &kenya
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

// Default returns the data embedded in the package, including boundaries
// and national figures. It is loaded on first use and shared afterwards.
func Default() *Data { return defaultData() }

// Load reads counties, wards and post offices in the JSON format of the
// files in the data directory. Boundaries are not loaded; add them with
// [Data.LoadWardShapes] and [Data.LoadCountyShapes].
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

// National returns the census figures for Kenya as a whole. It reports
// false for data from [Load], which has no national figures.
func (d *Data) National() (National, bool) {
	if d.national == nil {
		return National{}, false
	}
	return *d.national, true
}

// NationalLiving returns Kenya's living conditions, like [Data.National].
func (d *Data) NationalLiving() (Living, bool) {
	n, ok := d.National()
	return n.Living, ok
}

// Counties returns all 47 counties in code order.
func (d *Data) Counties() []County {
	return append([]County(nil), d.counties...)
}

// CountyByCode returns the county with a code from 1 to 47.
func (d *Data) CountyByCode(code int) (County, bool) {
	i, ok := d.countyByCode[code]
	if !ok {
		return County{}, false
	}
	return d.counties[i], true
}

// CountyByName returns the county with a name. Case, apostrophes, dashes
// and a trailing "County" don't matter, so "nakuru" and "Murang'a County"
// both work.
func (d *Data) CountyByName(name string) (County, bool) {
	i, ok := d.countyByName[normalize(name)]
	if !ok {
		return County{}, false
	}
	return d.counties[i], true
}

// NeighbouringCounties returns the counties that share a border with the
// county with code.
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

// WardByCode returns the ward with a four-digit IEBC code such as "0040".
func (d *Data) WardByCode(code string) (Ward, bool) {
	i, ok := d.wardByCode[code]
	if !ok {
		return Ward{}, false
	}
	return d.wards[i], true
}

// WardsInCounty returns the wards in a county, by ward code.
func (d *Data) WardsInCounty(countyCode int) []Ward {
	return d.pick(d.wardsByCounty[countyCode])
}

// Constituencies returns the names of a county's constituencies, sorted.
func (d *Data) Constituencies(countyCode int) []string {
	return append([]string(nil), d.constsByCounty[countyCode]...)
}

// WardsInConstituency returns the wards in the constituency with a name,
// matched like [Data.CountyByName].
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

// PostOffices returns the post offices with a postal code. Several offices
// can share one code.
func (d *Data) PostOffices(postalCode string) []PostOffice {
	return d.pickPost(d.postByCode[strings.TrimSpace(postalCode)])
}

// PostOfficesInCounty returns the post offices in a county, by postal code.
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
