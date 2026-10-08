# kenyageo

A Go package for Kenya's 47 counties, 289 constituencies (290 once the data is fixed, see below) and 1,450 wards. The data is compiled into the binary, so there is nothing to download at runtime.

```go
import kenyageo "github.com/rmiyoyo/kenya-geo" // alias, because the last path element has a hyphen

geo := kenyageo.Default()

c, ok := geo.CountyByName("murang'a")   // forgiving: case, apostrophes, dashes, "County" suffix
c, ok  = geo.CountyByCode(47)            // Nairobi
geo.Constituencies(c.Code)               // []string, alphabetical
geo.WardsInCounty(c.Code)                // []Ward
geo.WardsInConstituency("Kibra")         // []Ward
w, ok := geo.WardByCode("0040")          // WAA, Matuga, Kwale
geo.Search("nakru", 10)                  // fuzzy: counties, constituencies and wards, best first
```

## Try it

```sh
go test ./...                        # tests + runnable examples
go test -bench=. -benchmem           # benchmarks
go run ./cmd/kenyageo county nakuru  # constituencies and ward counts
go run ./cmd/kenyageo search kibera  # fuzzy search
```

The module path is `github.com/rmiyoyo/kenya-geo`; the package name is `kenyageo`, since Go package names cannot contain hyphens.

## Layout

| File | What it holds |
|---|---|
| `kenyageo.go` | Types (`County`, `Ward`, `Data`), embedding, `Load`, lookups |
| `search.go` | `Search`, name normalisation, Levenshtein distance |
| `kenyageo_test.go` | Table-driven tests, examples, benchmarks |
| `cmd/kenyageo/main.go` | A tiny CLI that uses the package like any outside caller would |
| `data/*.json` | Your two files, copied unmodified |

## Go concepts used, and where

- **`//go:embed`** (`kenyageo.go`): the compiler copies `data/*.json` into a `[]byte` at build time. Paths are relative to the source file and cannot leave the module.
- **Struct tags** like `` `json:"ward_code"` `` map JSON keys to Go fields. Only capitalised (exported) fields are decoded; `json:"-"` skips a field.
- **Exported vs unexported**: capitalised names (`County`, `Search`) are public; lowercase ones (`counties`, `normalize`) are private to the package. `Data` keeps its maps private so callers cannot corrupt them.
- **Pointers for "maybe missing"**: `RegisteredVoters2022 *int` is `nil` for the one ward whose value is `null`. A plain `int` would silently become 0.
- **"Comma ok"**: `CountyByCode` returns `(County, bool)`, the same pattern as reading a map (`v, ok := m[k]`). Go uses this instead of exceptions or null.
- **Errors are values**: `Load` returns `(*Data, error)`, and `fmt.Errorf("...: %w", err)` wraps the cause so callers can still inspect it with `errors.Is/As`.
- **Interfaces are implicit**: `Load` takes `io.Reader`, so a file, an HTTP body or `strings.NewReader` all work. `Kind` gets a `String()` method and thereby satisfies `fmt.Stringer` without declaring it.
- **`sync.OnceValue`**: `Default()` parses the JSON once, lazily, and is safe to call from many goroutines at the same time.
- **Range over a slice gives copies**: `Load` uses `w := &d.wards[i]` to modify wards in place; `for _, w := range d.wards` would modify a copy.
- **`iota`** builds the `KindCounty/KindConstituency/KindWard` enum.
- **Runes vs bytes**: Go strings are bytes. `levenshtein` converts to `[]rune` so `’` counts as one character, not three.
- **Testing**: table-driven tests with `t.Run`, `Example...` functions whose `// Output:` is checked by `go test`, and benchmarks with `b.Loop()` (Go 1.24).
- **Built-in `min`/`max`** (Go 1.21+) in `levenshtein` and `score`.

## Data issues found in the source files

These are reported, not fixed. The package keeps names exactly as given and only normalises them for matching.

**Counties file**
1. **Kilifi population is 109,735.** The 2009 census figure is 1,109,735, so a leading "1" looks dropped (2019 census: about 1.45M).
2. **Population year is unclear and probably mixed.** Several values match the 2009 census (Nairobi 3,138,369, Kiambu 1,623,282, Kakamega 1,660,651) while Kwale's 866,820 is its 2019 figure. Worth replacing the column with KNBS 2019 figures and naming the year in the field.
3. **Turkana's region is "North Eastern".** Turkana was in the former Rift Valley province; North Eastern was only Garissa, Wajir and Mandera.
4. **Regions mix two schemes.** "North Rift" sits alongside the former provinces ("Rift Valley", "Eastern", ...), so the Rift Valley is split for some counties but not others.
5. **Punctuation varies.** "Taita–Taveta" uses an en dash while "Tharaka-Nithi" and "Elgeyo-Marakwet" use hyphens; "Murang’a" uses a curly apostrophe. Matching handles all of these.

**Wards file**
1. **Gatundu North is missing.** Its four wards (GITUAMBA 0555, GITHOBOKONI 0556, CHANIA 0557, MANGU 0558) are labelled Gatundu South, so Kiambu shows 11 constituencies instead of 12 and Gatundu South shows 8 wards.
2. **DELLA (Eldas, Wajir) has code "01811810"**, the only code that is not 4 digits, and `registered_voters_2022` is `null`.
3. **"TAMB​UA" contains a hidden zero-width space**, so a plain string comparison with "TAMBUA" fails. Matching strips it.
4. **Two names are in mixed case**: "OMBeyi" and "OLOlMASANI"; every other ward name is upper case.
5. **33 ward names repeat across counties** (CENTRAL, TOWNSHIP, HOSPITAL, KALOLENI, ...), so a ward name alone is not an identifier; use the ward code.

## Ideas for your next Go steps

- `Search` takes about 1 ms and 8,000 allocations per call because it normalises every name on every query. Precomputing normalised names in `Load` is a good first optimisation; rerun the benchmark to measure it.
- Add an `http.Handler` in a new `api` package (step 3 of the plan).
- Add `go:generate` or a test that checks the data invariants above, so fixes to the JSON are verified.
