# Changelog

Versions follow [semantic versioning](https://semver.org/). Before 1.0, a minor version can add features; anything that would break existing code is called out under **Breaking**.

## v0.3.0 (2026-10-11)

Census data beyond population, and better post office locations.

### Added

- Living conditions from the 2019 census for every county: households, average household size, and the shares with mains electricity for lighting, piped water, internet use, a mobile phone and school attendance for ages 6 to 17. In Go, `County.Living`; in the API, a `living` object on every county.
- Urban and rural population for every county: `County.PopulationUrban`, `County.PopulationRural` and `County.UrbanPct`; `population_urban` and `population_rural` in the API.
- Religious affiliation for every county from the 2019 census: `County.Religion`, with `Religion.Christian` and `Religion.Pct`; a `religion` object in the API.
- The same census figures for Kenya as a whole: `Data.National` and `Data.NationalLiving`, from `data/kenya.json`.
- `GET /national` in the API returns Kenya's totals: population by sex, area, urban and rural population, living conditions, religion and registered voters.
- Post office locations from OpenStreetMap where GeoNames has none, so 577 of the 890 offices are now placed. `location_source` says which source placed each one: `geonames`, `openstreetmap`, or empty.
- The API reference in [`api/README.md`](api/README.md), with an example for every endpoint. It is embedded in the package as `api.Docs`, so websites can show docs that match the version they serve. Tests check that every route is documented and every example works.

### Changed

- County responses have new fields: `population_urban`, `population_rural`, `living` and `religion`. Code that reads county JSON keeps working; strict decoders that reject unknown fields need the new fields added.
- `api/openapi.json` describes the new fields and is now version 0.3.0.
- `location_source` can now be `openstreetmap`.

### Removed

- `docs/API.md`. The reference moved to `api/README.md`.

## v0.2.0 (2026-10-11)

- A JSON HTTP API in the `api` package and `kenyageo-server` to run it, with an OpenAPI 3.1 description and a Docker image.
- `WardAt` and `CountyAt` to find the ward or county for a coordinate.
- `PostOfficesNear` to find the closest post offices to a point.
- Neighbouring counties, ward and county boundaries as GeoJSON, and registered voter totals for counties and constituencies.
- A parser for Kenyan postal addresses.
- Search about six times faster.
- `-json` output for the `kenyageo` command.
- Fuzz tests, staticcheck in CI, and a CI check that the generated data matches the generator.

## v0.1.0 (2026-10-09)

- The `kenyageo` package with embedded data for the 47 counties, their constituencies and 1,450 wards, from the KNBS 2019 census, the IEBC and geoBoundaries.
- 890 post offices from GeoNames, each placed in a county.
