# kenya-geo

[![Go Reference](https://pkg.go.dev/badge/github.com/rmiyoyo/kenya-geo.svg)](https://pkg.go.dev/github.com/rmiyoyo/kenya-geo)
[![CI](https://github.com/rmiyoyo/kenya-geo/actions/workflows/ci.yml/badge.svg)](https://github.com/rmiyoyo/kenya-geo/actions/workflows/ci.yml)

Kenya's 47 counties, 290 constituencies, 1,450 wards and 890 post offices for Go, with 2019 census figures and map coordinates. The data is compiled into your binary, so there is nothing to download or configure at runtime.

## Features

- Every county, constituency and ward, with IEBC ward codes and 2022 registered voters
- 2019 census population by sex, land area and density for each county
- ISO 3166-2 codes, former provinces and neighbouring counties
- Centroids and bounding boxes for counties, centroids for wards
- Find the ward, constituency and county for any coordinate, or just the county from county boundaries
- Ward and county boundaries as GeoJSON
- 890 post offices by postal code, each placed in a county, and the nearest offices to any point
- A postal address parser that checks the box, postal code and town agree
- Forgiving name lookups: case, apostrophes, dashes and a trailing "County" don't matter
- Fuzzy search across counties, constituencies, wards and post offices
- A JSON HTTP API and a ready-to-run server
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
geo.NeighbouringCounties(47)   // Machakos, Kiambu and Kajiado
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

`CountyAt` does the same with county boundaries, so it also works where no ward boundary is known:

```go
c, ok := geo.CountyAt(-0.0917, 34.768)  // Kisumu County
```

The two use different boundary sets, so within a few kilometres of a county line they can disagree: the ward's `CountyName` follows the IEBC ward list, while `CountyAt` follows the county outline.

### Boundaries as GeoJSON

`WardBoundary` and `CountyBoundary` return the outline as a GeoJSON Feature, ready for Leaflet, Mapbox or QGIS:

```go
f, ok := geo.WardBoundary("1439")
b, _ := json.Marshal(f)
// {"type":"Feature","geometry":{"type":"MultiPolygon","coordinates":[...]},"properties":{"ward_code":"1439",...}}
```

The geometry is always a `MultiPolygon` with closed rings, outer rings counterclockwise and holes clockwise, as RFC 7946 asks. `Properties` is the `Ward` or `County`, so the JSON carries the same fields as the other lookups. Each call returns a fresh copy, so changing it does not affect later lookups. Wards without a known boundary return false.

### Post offices

```go
geo.PostOffices("00100")       // Nairobi GPO
geo.PostOfficesInCounty(27)    // every post office in Uasin Gishu
```

A few postal codes are shared by two offices, so `PostOffices` returns a slice.

```go
for _, p := range geo.PostOfficesNear(-0.2833, 36.0667, 3) {
    fmt.Println(p.Code, p.Name, p.DistanceKm)  // 20100 Nakuru 2.7, then Lanet, Kabatini
}
```

`PostOfficesNear` returns the closest post offices to a point, nearest first, with the straight-line distance in kilometres. Only the 553 offices with a known location are considered, so the nearest result can be a few kilometres off where an office without a location (such as Nairobi GPO) is closer.

### Postal addresses

`ParseAddress` reads a Kenyan postal address and checks it against the post office list:

```go
a, err := geo.ParseAddress("P.O. Box 30100, 00100 GPO Nairobi")
fmt.Println(a.Box, a.PostalCode, a.PostOffice.Name) // 30100 00100 Nairobi Gpo
fmt.Println(a)                                      // P.O. Box 30100-00100 Nairobi
```

It accepts the usual ways of writing one: `P.O. Box`, `P. O. BOX`, `PO Box`, `Box` and `Private Bag`, with the box number and postal code joined by a dash, a comma or a space, in any case, with a name on the lines before and "Kenya" at the end. When the postal code is missing it is taken from the town, so `Box 45, Kisumu` becomes `P.O. Box 45-40100 Kisumu`.

The town is checked against the post office for that code. It matches when it is the office's name (one typo allowed) or the county the office is in, so `P.O. Box 123-00101 Nairobi` is fine even though 00101 is the Jamia office. `String` prints the address in the standard `P.O. Box 123-00100 Nairobi` form.

Errors can be checked with `errors.Is`:

| Error | Meaning |
| --- | --- |
| `ErrNoBox` | no P.O. Box or Private Bag in the text, such as a street address |
| `ErrUnknownPostOffice` | the postal code doesn't exist, or there is no code and the town isn't a post office |
| `ErrTownMismatch` | the town doesn't belong to the postal code; the address is still returned, with the office for the code |

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
err = geo.LoadWardShapes(wardShapesFile)     // needed for WardAt and WardBoundary
err = geo.LoadCountyShapes(countyShapesFile) // needed for CountyAt and CountyBoundary
```

## Command-line tool

```sh
go install github.com/rmiyoyo/kenya-geo/cmd/kenyageo@latest

kenyageo county nakuru    # constituencies and ward counts
kenyageo postcode 30100   # post office lookup
kenyageo search kibera    # fuzzy search
kenyageo at -1.2884 36.8233  # ward, constituency and county for a point
kenyageo near -0.2833 36.0667  # the five closest post offices
kenyageo boundary 1439       # a ward or county outline as GeoJSON
kenyageo address "P.O. Box 123-00100 Nairobi"  # parse and check a postal address
```

## HTTP API

The `api` package serves the same data as JSON, and `kenyageo-server` runs it:

```sh
go install github.com/rmiyoyo/kenya-geo/cmd/kenyageo-server@latest
kenyageo-server -addr :8080
```

| Endpoint | Returns |
| --- | --- |
| `GET /counties` | all 47 counties |
| `GET /counties/{county}` | one county, by code (`32`) or name (`nakuru`) |
| `GET /counties/{county}/constituencies` | constituency names |
| `GET /counties/{county}/neighbours` | counties that share a border with it |
| `GET /counties/{county}/boundary` | the county outline as a GeoJSON Feature |
| `GET /counties/{county}/wards` | wards in the county |
| `GET /counties/{county}/postoffices` | post offices in the county |
| `GET /constituencies/{name}/wards` | wards in a constituency, e.g. `/constituencies/kibra/wards` |
| `GET /wards/{code}` | one ward |
| `GET /wards/{code}/boundary` | the ward outline as a GeoJSON Feature |
| `GET /postcodes/{code}` | post offices with that postal code |
| `GET /search?q=nakru&limit=10` | fuzzy search results with `kind`, `name`, `code`, `county_code` and `score` |
| `GET /addresses?q=P.O.%20Box%20123-00100%20Nairobi` | the parsed address with its post office; 400 if there is no box, 404 for an unknown office, 422 if the town and code disagree |
| `GET /at?lat=-1.2884&lng=36.8233` | the ward at that point |
| `GET /counties/at?lat=-0.0917&lng=34.768` | the county at that point |
| `GET /postoffices/near?lat=-0.2833&lng=36.0667&limit=5` | the closest post offices, nearest first, with `distance_km` |

Boundary responses use the `application/geo+json` content type. Errors come back as `{"error": "..."}` with status 400 for a bad query and 404 when nothing matches. Responses allow requests from any origin, so a web page can call the API directly.

To add the endpoints to your own server, mount the handler:

```go
mux.Handle("/geo/", http.StripPrefix("/geo", api.New(kenyageo.Default())))
```

## Data

| File | Contents |
| --- | --- |
| `data/counties.json` | code, name, headquarters, ISO code, former province, population by sex, area, centroid, bounding box, neighbouring county codes |
| `data/wards.json` | ward code, name, constituency, county, 2022 registered voters, centroid |
| `data/postoffices.json` | postal code, name, county, how the county was decided, location |
| `data/wardshapes.json` | ward code and boundary polygons, as GeoJSON-style `[lng, lat]` rings |
| `data/countyshapes.json` | county code and boundary polygons, in the same format |

The same JSON files can be used from any language.

### Accuracy notes

- Neighbouring counties are worked out from the county boundaries: two counties are neighbours when their outlines run within about a kilometre of each other for at least three boundary points. Borders across Lake Victoria count, so Siaya and Homa Bay are neighbours.
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
