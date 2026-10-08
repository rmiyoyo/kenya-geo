# kenya-geo

A Go package for Kenya's 47 counties, 290 constituencies, 1,450 wards and 890 post offices. It includes 2019 census figures and map coordinates. The data is compiled into the binary, so nothing is downloaded at runtime.

```go
import kenyageo "github.com/rmiyoyo/kenya-geo"

geo := kenyageo.Default()

c, ok := geo.CountyByName("murang'a")   // forgiving: case, apostrophes, dashes, "County" suffix
c, ok  = geo.CountyByCode(47)            // Nairobi
c.Population, c.Density(), c.ISOCode     // 4397073, 6247/km², "KE-30"
geo.Constituencies(c.Code)               // []string, alphabetical
geo.WardsInCounty(c.Code)                // []Ward
geo.WardsInConstituency("Kibra")         // []Ward
w, ok := geo.WardByCode("0040")          // WAA, Matuga, Kwale
w.Centroid                               // *LatLng, nil when unknown
geo.PostOffices("00100")                 // []PostOffice: Nairobi GPO
geo.PostOfficesInCounty(27)              // post offices in Uasin Gishu
geo.Search("nakru", 10)                  // fuzzy: counties, constituencies, wards and post offices
```

## Try it

```sh
go test ./...                        # tests + runnable examples
go test -bench=. -benchmem           # benchmarks
go run ./cmd/kenyageo county nakuru  # constituencies and ward counts
go run ./cmd/kenyageo search kibera  # fuzzy search
go run ./cmd/kenyageo postcode 30100 # post office lookup
go run ./internal/gendata            # rebuild data/*.json from the sources
```

## What's in the data

**Counties** (`data/counties.json`)

| Field | Source |
|---|---|
| `code`, `name`, `headquarters` | original counties file |
| `iso_code` (ISO 3166-2, e.g. `KE-29`) | geoBoundaries |
| `former_province` (pre-2010 province) | derived from the county code |
| `population`, `population_male`, `population_female`, `population_intersex`, `area_km2` | KNBS 2019 census, Volume I |
| `centroid`, `bbox` | computed from geoBoundaries polygons |

ISO codes are numbered alphabetically, so they don't match the county codes (Turkana is county 23 but `KE-43`).

**Wards** (`data/wards.json`): `ward_code`, `name`, `constituency_name`, `county_code`, `county_name` and `registered_voters_2022` come from the original IEBC wards file. `centroid` is computed from geoBoundaries polygons and is known for 1,432 of 1,450 wards.

**Post offices** (`data/postoffices.json`): `postal_code`, `name`, `county_code`, `county_source` and `location`, from the GeoNames postal code file for Kenya. A few codes are shared by two offices, so `PostOffices` returns a slice. `county_source` says where the county came from: `geonames`, `location`, `manual` or `neighbours` (see below); every office now has a county. `location` is set only for the 553 offices GeoNames matched to a real place; for the other 337 its coordinates are averages of neighbouring codes, often in the wrong county, so they're left out.

Centroids and bounding boxes are good for placing map labels or zooming a map. They can't tell you which ward a point is in; that needs the polygons, which is a natural next step.

## Sources

| Source | Used for | Licence |
|---|---|---|
| [KNBS 2019 Kenya Population and Housing Census, Volume I](https://open.africa/dataset/2019-kenya-population-and-housing-census) (via openAFRICA) | county population by sex, land area | open data |
| [geoBoundaries gbOpen KEN ADM1/ADM3](https://www.geoboundaries.org/) (RCMRD source, 2020) | ISO codes, centroids, bounding boxes | public domain |
| [GeoNames postal codes](https://www.geonames.org/) (`data/source/geonames-KE.txt`, from KE.zip) | post offices | CC BY 4.0, credit GeoNames |
| `data/source/counties.json`, `data/source/wards.json` | codes, names, HQs, wards, 2022 voters | your original files |

`go run ./internal/gendata` downloads the KNBS and geoBoundaries files into `data/raw/` (git-ignored), merges them with `data/source/`, applies the fixes below and writes `data/counties.json` and `data/wards.json`. Never edit those two files by hand.

## Fixes applied by the generator

| Problem in the source | Fix | How it was checked |
|---|---|---|
| Gatundu North missing: wards 0555–0558 (GITUAMBA, GITHOBOKONI, CHANIA, MANGU) labelled Gatundu South | Set to Gatundu North | All four lie inside Gatundu North in the geoBoundaries polygons; Kiambu now has 12 constituencies, Kenya 290 |
| DELLA had code `01811810` | `0181` | Sits between ELDAS (0180) and LAKOLEY SOUTH/BASIR (0182) in the IEBC numbering |
| Kilifi population 109,735 | KNBS 2019: 1,453,787 | All populations replaced; they sum to the national 47,564,296 |
| Populations mixed 2009 and 2019 census figures | All KNBS 2019 | Same |
| Turkana in "North Eastern"; "North Rift" mixed with provinces | `former_province`, from code ranges, 8 provinces | Turkana is Rift Valley |
| "Taita–Taveta" (en dash), "Murang’a" (curly apostrophe) | ASCII `-` and `'` | Name matching still accepts both |
| Zero-width space inside TAMBUA | Removed | |
| "OMBeyi", "OLOlMASANI" in mixed case | Upper-cased | |
| KNBS sub-county table: Nakuru male 177,272 and female 184,835 | 1,077,272 and 1,084,835 | Male + female + intersex now equals the county total 2,162,202 |
| GeoNames puts offices it can't place in Nairobi, e.g. Koracha (40639) and Singore (30703), far outside the city | If GeoNames matched the office to a real place, the county comes from its location (Singore → Elgeyo-Marakwet). Otherwise it comes from a manual fix, or from the post offices whose codes share its first four digits, when at least three of them agree and they make up at least 75% (Koracha 40639 → Siaya, from 5 of 5). Nairobi head offices (00100, 00200 ... 00800) stay in Nairobi | Location tested against the geoBoundaries Nairobi outline; manual fixes (`postOfficeCountyFixes` in `internal/gendata/fixes.go`): Kenyatta Hospital → Nairobi, Bunyore → Vihiga, Budalangi, Kakemer and Kocholya → Busia, Kakibora → Trans-Nzoia, Nthogoini → Machakos |

Known remaining gaps:
- DELLA still has no 2022 voter count.
- 18 wards have no centroid because their names differ too much between IEBC and geoBoundaries (e.g. HIRIMANI, MKOMANI and the two TOWNSHIP wards in Kiambu).
- The geoBoundaries county outline puts four Bureti wards (TEBESONIK, CHEBOIN, CHEMOSOT, LITEIN) just across the Kericho–Bomet line; their centroids are right, the county outline is slightly off.
- 33 ward names repeat across counties (CENTRAL, TOWNSHIP, ...), so use ward codes as identifiers.
- 23 post office counties are inferred from neighbouring codes rather than confirmed; `county_source` is `neighbours` for those. Outside Nairobi, GeoNames' county also disagrees with the map for about 120 offices. Spot checks found GeoNames' county right more often than the map test (Kimana is in Kajiado, Hola in Tana River), so its county is kept.

## Layout

| Path | What it holds |
|---|---|
| `kenyageo.go` | Types, embedding, `Load`, lookups |
| `search.go` | `Search`, name normalisation, Levenshtein distance |
| `kenyageo_test.go` | Table-driven tests, examples, benchmarks |
| `cmd/kenyageo/` | A small CLI that uses the package like an outside caller |
| `internal/gendata/` | The data generator: downloads, census parsing, point-in-polygon, fixes |
| `data/source/` | Your original files and the GeoNames file, unmodified |
| `data/*.json` | Generated, embedded files |

## Go concepts used, and where

- **`//go:embed`** (`kenyageo.go`): the compiler copies `data/*.json` into a `[]byte` at build time. Paths are relative to the source file and cannot leave the module.
- **Struct tags** like `` `json:"ward_code"` `` map JSON keys to Go fields. Only capitalised (exported) fields are decoded.
- **Exported vs unexported**: capitalised names (`County`, `Search`) are public; lowercase ones (`counties`, `normalize`) are private to the package. `Data` keeps its maps private so callers can't corrupt them. `internal/` goes further: other modules can't import anything under it at all.
- **Pointers for "maybe missing"**: `RegisteredVoters2022 *int` and `Centroid *LatLng` are `nil` when unknown. A plain value would silently become zero.
- **Methods and receivers**: `County.Density()` has a value receiver, so it works on a copy. `Load` writes through `w := &d.wards[i]`, because `for _, w := range d.wards` would give a copy.
- **"Comma ok"**: `CountyByCode` returns `(County, bool)`, the same pattern as reading a map (`v, ok := m[k]`).
- **Errors are values**: `Load` returns `(*Data, error)`, and `fmt.Errorf("...: %w", err)` wraps the cause so callers can inspect it with `errors.Is/As`.
- **Interfaces are implicit**: `Load` takes `io.Reader`, so a file, an HTTP body or `strings.NewReader` all work. `Kind` has a `String()` method and thereby satisfies `fmt.Stringer`.
- **`sync.OnceValue`**: `Default()` parses the JSON once, lazily, and is safe to call from many goroutines.
- **`iota`** builds the `KindCounty/KindConstituency/KindWard` enum.
- **Runes vs bytes**: Go strings are bytes. `levenshtein` converts to `[]rune` so `’` counts as one character, not three.
- **Generics** (`internal/gendata/main.go`): `writeLines[T any]` writes any slice as one JSON object per line, so a change to one ward is a one-line git diff.
- **`json.RawMessage`** (`internal/gendata/geo.go`): GeoJSON nests Polygon and MultiPolygon coordinates differently, so the coordinates are kept raw until the geometry type is known.
- **`strings.Map`** (`internal/gendata/fixes.go`): returning a negative rune drops a character, which is how the zero-width space is removed.
- **Ray casting** (`internal/gendata/geo.go`): a point is inside a polygon if a line from it crosses the boundary an odd number of times. The generator uses this to find which county each ward polygon belongs to.
- **Testing**: table-driven tests with `t.Run`, `Example...` functions whose `// Output:` is checked by `go test`, and benchmarks with `b.Loop()` (Go 1.24).

## Ideas for your next Go steps

- `Search` takes about 1 ms and 8,000 allocations per call because it normalises every name on every query. Precompute normalised names in `Load` and rerun the benchmark.
- Embed simplified ward polygons and add `WardAt(lat, lng)` using the ray-casting code from the generator.
- Add an `http.Handler` in a new `api` package (step 3 of the plan).
