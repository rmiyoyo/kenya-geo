// Package kenyageo gives typed, in-memory access to Kenya's 47 counties,
// their constituencies and their 1,450 electoral wards.
//
// The data ships inside the compiled binary (see go:embed below), so there
// are no files to deploy and no network calls. Most callers only need
// Default:
//
//	geo := kenyageo.Default()
//	nairobi, ok := geo.CountyByName("nairobi")
//	wards := geo.WardsInCounty(nairobi.Code)
package kenyageo

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
)

// The //go:embed directive tells the compiler to copy a file's bytes into
// the variable below at build time. The path is relative to this source
// file and may not reach outside the module (no "..").

//go:embed data/counties.json
var countiesJSON []byte

//go:embed data/wards.json
var wardsJSON []byte

// County is one of Kenya's 47 counties. The struct tags (`json:"..."`) map
// the JSON keys onto Go's exported (capitalised) field names.
type County struct {
	Code         int     `json:"code"`
	Name         string  `json:"name"` // e.g. "Murang’a County", kept exactly as in the source
	Region       string  `json:"region"`
	Headquarters string  `json:"headquarters"`
	Population   int     `json:"population"`
	AreaKm2      float64 `json:"area_km2"`
}

// Ward is an electoral ward (the smallest unit in the data).
type Ward struct {
	Code         string `json:"ward_code"` // a string, so leading zeros like "0040" survive
	Name         string `json:"name"`
	Constituency string `json:"constituency_name"`
	CountyName   string `json:"county_name"`

	// RegisteredVoters2022 is a pointer because the source has a null for one
	// ward. nil means "unknown", which a plain int (zero) could not express.
	RegisteredVoters2022 *int `json:"registered_voters_2022"`

	// CountyCode is not in the ward JSON; Load fills it in by matching
	// CountyName against the counties file. The "-" tag tells encoding/json
	// to ignore it when decoding.
	CountyCode int `json:"-"`
}

// Data holds the parsed datasets plus the indexes that make lookups O(1).
// Its fields are unexported (lowercase), so callers can only reach the data
// through methods, and cannot mutate the shared maps by accident.
type Data struct {
	counties []County // sorted by Code
	wards    []Ward   // sorted by Code

	countyByCode   map[int]int    // county code -> index into counties
	countyByName   map[string]int // normalized name -> index into counties
	wardByCode     map[string]int // ward code -> index into wards
	wardsByCounty  map[int][]int  // county code -> indexes into wards
	wardsByConst   map[string][]int
	constsByCounty map[int][]string // county code -> constituency names, sorted
	constCounty    map[string]int   // normalized constituency -> county code
}

// defaultData parses the embedded JSON once, the first time it is needed.
// sync.OnceValue makes this safe even if many goroutines call Default at
// the same moment.
var defaultData = sync.OnceValue(func() *Data {
	d, err := Load(bytes.NewReader(countiesJSON), bytes.NewReader(wardsJSON))
	if err != nil {
		// The embedded files are part of the build and covered by tests, so
		// failing here is a programming error rather than a runtime condition.
		panic("kenyageo: embedded data is invalid: " + err.Error())
	}
	return d
})

// Default returns the dataset bundled with this package.
func Default() *Data { return defaultData() }

// Load parses county and ward JSON from any readers (files, HTTP bodies,
// test strings) and builds the lookup indexes. Taking io.Reader instead of
// a file path keeps the function easy to test and reuse.
func Load(counties, wards io.Reader) (*Data, error) {
	d := &Data{}
	if err := json.NewDecoder(counties).Decode(&d.counties); err != nil {
		return nil, fmt.Errorf("decoding counties: %w", err)
	}
	if err := json.NewDecoder(wards).Decode(&d.wards); err != nil {
		return nil, fmt.Errorf("decoding wards: %w", err)
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

	// Index into the slice (d.wards[i]) rather than using the loop variable
	// w, because w is a copy and we need to write CountyCode back.
	for i := range d.wards {
		w := &d.wards[i]
		ci, ok := d.countyByName[normalize(w.CountyName)]
		if !ok {
			return nil, fmt.Errorf("ward %s (%s): unknown county %q", w.Code, w.Name, w.CountyName)
		}
		if _, dup := d.wardByCode[w.Code]; dup {
			return nil, fmt.Errorf("duplicate ward code %s", w.Code)
		}
		w.CountyCode = d.counties[ci].Code
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
	return d, nil
}

// Counties returns all counties ordered by code. It returns a copy, so the
// caller may sort or modify the slice without affecting the Data.
func (d *Data) Counties() []County {
	return append([]County(nil), d.counties...)
}

// CountyByCode looks a county up by its official code (1 = Mombasa ...
// 47 = Nairobi). The second result reports whether it was found; this
// "comma ok" pattern is how Go signals "not found" without an error.
func (d *Data) CountyByCode(code int) (County, bool) {
	i, ok := d.countyByCode[code]
	if !ok {
		return County{}, false
	}
	return d.counties[i], true
}

// CountyByName finds a county by name, forgiving case, a missing or extra
// "County" suffix, and punctuation differences: "muranga", "Murang'a" and
// "MURANG’A COUNTY" all match, as do "Taita Taveta" and "Taita-Taveta".
func (d *Data) CountyByName(name string) (County, bool) {
	i, ok := d.countyByName[normalize(name)]
	if !ok {
		return County{}, false
	}
	return d.counties[i], true
}

// WardByCode looks a ward up by its IEBC code, e.g. "0040".
func (d *Data) WardByCode(code string) (Ward, bool) {
	i, ok := d.wardByCode[code]
	if !ok {
		return Ward{}, false
	}
	return d.wards[i], true
}

// WardsInCounty returns the county's wards ordered by ward code, or nil if
// the code is unknown.
func (d *Data) WardsInCounty(countyCode int) []Ward {
	return d.pick(d.wardsByCounty[countyCode])
}

// Constituencies returns the names of the county's constituencies,
// alphabetically.
func (d *Data) Constituencies(countyCode int) []string {
	return append([]string(nil), d.constsByCounty[countyCode]...)
}

// WardsInConstituency returns the wards of a constituency, matched by name
// with the same forgiving rules as CountyByName. Constituency names are
// unique nationwide, so no county is needed.
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
