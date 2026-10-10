# kenya-geo

[![Go Reference](https://pkg.go.dev/badge/github.com/rmiyoyo/kenya-geo.svg)](https://pkg.go.dev/github.com/rmiyoyo/kenya-geo)
[![CI](https://github.com/rmiyoyo/kenya-geo/actions/workflows/ci.yml/badge.svg)](https://github.com/rmiyoyo/kenya-geo/actions/workflows/ci.yml)

Kenya's 47 counties, 290 constituencies, 1,450 wards and 890 post offices for Go, with 2019 census figures and map coordinates. The data is compiled into your binary, so there is nothing to download or configure at runtime.

## Features

- Every county, constituency and ward, with IEBC ward codes and 2022 registered voters
- 2019 census population by sex, land area and density for each county
- ISO 3166-2 codes and former provinces
- Centroids and bounding boxes for counties, centroids for wards
- Find the ward, constituency and county for any coordinate
- 890 post offices by postal code, each placed in a county
- Forgiving name lookups: case, apostrophes, dashes and a trailing "County" don't matter
- Fuzzy search across counties, constituencies, wards and post offices
- No dependencies outside the standard library; safe for concurrent use

## Install

```sh
go get github.com/rmiyoyo/kenya-geo
```

Requires Go 1.24 or later.

## Usage

```go
package main

import (
    "fmt"

    kenyageo "github.com/rmiyoyo/kenya-geo"
)

func main() {
    geo := kenyageo.Default()

    c, ok := geo.CountyByName("murang'a")
    if !ok {
        return
    }
    fmt.Println(c.Code, c.Name, c.Headquarters) // 21 Murang'a County Murang'a

    for _, name := range geo.Constituencies(c.Code) {
        fmt.Println(name, len(geo.WardsInConstituency(name)))
    }
}
```

### Counties

```go
geo.Counties()                 // all 47, ordered by county code
geo.CountyByCode(47)           // Nairobi County
geo.CountyByName("nairobi")    // the same county
c.Population, c.Density()      // 4397073, about 6247 people per km²
c.ISOCode, c.FormerProvince    // "KE-30", "Nairobi"
```

### Constituencies and wards

```go
geo.Constituencies(32)         // Nakuru's 11 constituencies, alphabetical
geo.WardsInCounty(32)          // []Ward
geo.WardsInConstituency("Kibra")
geo.WardByCode("0040")         // WAA, in Matuga, Kwale
w.Centroid                     // *LatLng, nil when unknown
w.RegisteredVoters2022         // *int, nil when unknown
```

Ward names repeat across the country (there are several CENTRAL and TOWNSHIP wards), so store ward codes, not names.

### Which ward is this point in?

```go
w, ok := geo.WardAt(-1.2884, 36.8233)  // latitude, longitude
fmt.Println(w.Name, w.Constituency, w.CountyName)
// NAIROBI CENTRAL Starehe Nairobi County
```

`WardAt` returns false for points outside Kenya, in the sea, or in one of the few areas without a ward boundary (see the accuracy notes). The boundaries are loaded the first time you call it, which takes about 30 ms; after that a lookup takes a few microseconds.

### Post offices

```go
geo.PostOffices("00100")       // Nairobi GPO
geo.PostOfficesInCounty(27)    // every post office in Uasin Gishu
```

A few postal codes are shared by two offices, so `PostOffices` returns a slice.

### Search

```go
for _, m := range geo.Search("nakru", 10) {
    fmt.Println(m.Kind, m.Name, m.Score)
}
```

Results are ranked by how closely they match, and each carries the code of the county it belongs to.

### Your own data

`Load` builds the same lookups from any readers, for example a newer copy of the files in `data/`:

```go
geo, err := kenyageo.Load(countiesFile, wardsFile, postOfficesFile)
err = geo.LoadWardShapes(wardShapesFile)   // needed for WardAt
```

## Command-line tool

```sh
go install github.com/rmiyoyo/kenya-geo/cmd/kenyageo@latest

kenyageo county nakuru    # constituencies and ward counts
kenyageo postcode 30100   # post office lookup
kenyageo search kibera    # fuzzy search
kenyageo at -1.2884 36.8233  # ward, constituency and county for a point
```

## Data

| File | Contents |
| --- | --- |
| `data/counties.json` | code, name, headquarters, ISO code, former province, population by sex, area, centroid, bounding box |
| `data/wards.json` | ward code, name, constituency, county, 2022 registered voters, centroid |
| `data/postoffices.json` | postal code, name, county, how the county was decided, location |
| `data/wardshapes.json` | ward code and boundary polygons, as GeoJSON-style `[lng, lat]` rings |

The same JSON files can be used from any language.

### Accuracy notes

- Ward centroids and boundaries are known for 1,437 of the 1,450 wards. The other 13 have names too different from the boundary data to match safely, so `WardAt` finds nothing in those areas.
- Boundaries are simplified, so a point within a few hundred metres of a ward line can land in the neighbouring ward. A handful of wards on county borders have centroids that fall just across the line.
- Post office locations are given only for the 553 offices matched to a real place. `county_source` says how each office's county was decided: `geonames`, `location`, `manual` or `neighbours`.
- The source data had several errors, such as wards filed under the wrong constituency, mixed census years and a malformed ward code. They are corrected when the data is generated; see `internal/gendata/fixes.go`.

## Sources

| Source | Used for | Licence |
| --- | --- | --- |
| [KNBS 2019 Kenya Population and Housing Census, Volume I](https://open.africa/dataset/2019-kenya-population-and-housing-census) | county population by sex, land area | open data |
| [geoBoundaries gbOpen KEN ADM1/ADM2/ADM3](https://www.geoboundaries.org/) | ISO codes, centroids, bounding boxes, ward boundaries | public domain |
| [GeoNames postal codes](https://www.geonames.org/) | post offices | CC BY 4.0 |
| IEBC county and ward lists | codes, names, headquarters, wards, 2022 voters | public record |

Post office data is © GeoNames, used under CC BY 4.0.

## Contributing

Issues and pull requests are welcome. To regenerate the data after changing a source or a fix:

```sh
go run ./internal/gendata   # downloads raw sources, applies fixes, writes data/*.json
go test ./...
```

Don't edit `data/*.json` by hand. Change the sources in `data/source/` or the fixes in `internal/gendata/` instead.
