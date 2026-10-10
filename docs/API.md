# HTTP API

The `api` package serves Kenya's counties, constituencies, wards and post offices as JSON. Every endpoint is a `GET`, needs no key, and allows requests from any origin, so a web page can call it directly.

Run it yourself with `kenyageo-server`:

```sh
go install github.com/rmiyoyo/kenya-geo/cmd/kenyageo-server@latest
kenyageo-server -addr :8080
```

Or mount it inside your own Go server:

```go
mux.Handle("/geo/", http.StripPrefix("/geo", api.New(kenyageo.Default())))
```

The paths below are relative to wherever the API is mounted.

## Conventions

- Responses are JSON with `Content-Type: application/json`. Boundaries use `application/geo+json`.
- Errors come back as `{"error": "..."}` with status 400 for a bad query and 404 when nothing matches.
- A county can be given by code (`32`) or by name (`nakuru`, `Nakuru County`). Case, apostrophes, dashes and a trailing "County" don't matter.
- Constituency names are forgiving in the same way. Encode a slash in a name as `%2F`.
- Ward codes are the IEBC's four-digit codes, such as `0040`. Postal codes have five digits, such as `00100`.
- Coordinates are decimal degrees in `lat` and `lng`.
- `limit` takes a number from 1 to 100.
- `GET /openapi.json` describes every endpoint in OpenAPI 3.1, for Swagger UI, Postman or a client generator.

## Counties

### GET /counties

All 47 counties, ordered by county code.

```sh
curl localhost:8080/counties
```

### GET /counties/{county}

One county, with 2019 census figures, its ISO 3166-2 code, its centroid and bounding box, and the codes of its neighbours.

```sh
curl localhost:8080/counties/47
```

```json
{
  "code": 47,
  "name": "Nairobi County",
  "iso_code": "KE-30",
  "former_province": "Nairobi",
  "headquarters": "Nairobi",
  "population": 4397073,
  "population_male": 2192452,
  "population_female": 2204376,
  "population_intersex": 245,
  "area_km2": 703.9,
  "centroid": {"lat": -1.29275, "lng": 36.8644},
  "bbox": [36.65987, -1.44406, 37.10452, -1.16046],
  "neighbours": [16, 22, 34]
}
```

`bbox` is `[west, south, east, north]`. Returns 404 for an unknown county.

### GET /counties/{county}/constituencies

The county's constituency names, alphabetical.

```sh
curl localhost:8080/counties/nairobi/constituencies
```

```json
["Dagoretti North", "Dagoretti South", "Embakasi Central", "..."]
```

### GET /counties/{county}/neighbours

The counties that share a border with it, as full county objects. Borders across Lake Victoria count.

```sh
curl localhost:8080/counties/47/neighbours
```

### GET /counties/{county}/wards

Every ward in the county, as ward objects. See [`GET /wards/{code}`](#get-wardscode) for the shape.

```sh
curl localhost:8080/counties/mombasa/wards
```

### GET /counties/{county}/postoffices

Every post office filed under the county. See [`GET /postcodes/{code}`](#get-postcodescode) for the shape.

```sh
curl localhost:8080/counties/1/postoffices
```

### GET /counties/{county}/voters

Registered voters in 2022, summed over the county's wards.

```sh
curl localhost:8080/counties/47/voters
```

```json
{"registered_voters_2022": 2415310, "wards": 85, "complete": true}
```

`complete` is false when a ward in the total has no figure. Only Della ward in Eldas, Wajir is missing one.

### GET /counties/{county}/boundary

The county outline as a GeoJSON Feature with a MultiPolygon geometry. The county object is in `properties`.

```sh
curl localhost:8080/counties/32/boundary
```

### GET /counties/at

The county containing a point, from the county boundaries.

| Parameter | Required | Meaning |
| --- | --- | --- |
| `lat` | yes | latitude |
| `lng` | yes | longitude |

```sh
curl "localhost:8080/counties/at?lat=-0.0917&lng=34.768"
```

Returns the county object, 400 if `lat` or `lng` isn't a number, and 404 for a point outside Kenya.

## Constituencies

### GET /constituencies/{name}/wards

The wards in a constituency.

```sh
curl localhost:8080/constituencies/kibra/wards
```

Returns 404 for an unknown constituency.

### GET /constituencies/{name}/voters

Registered voters in 2022, summed over the constituency's wards.

```sh
curl localhost:8080/constituencies/kibra/voters
```

```json
{"registered_voters_2022": 128282, "wards": 5, "complete": true}
```

## Wards

### GET /wards/{code}

One ward, by IEBC ward code.

```sh
curl localhost:8080/wards/0040
```

```json
{
  "ward_code": "0040",
  "name": "WAA",
  "constituency_name": "Matuga",
  "county_code": 2,
  "county_name": "Kwale County",
  "registered_voters_2022": 20681,
  "centroid": {"lat": -4.16191, "lng": 39.58915}
}
```

`registered_voters_2022` and `centroid` are null when unknown. Returns 404 for an unknown code.

### GET /wards/{code}/boundary

The ward outline as a GeoJSON Feature, with the ward object in `properties`. Returns 404 for the 13 wards whose boundary isn't known.

```sh
curl localhost:8080/wards/0040/boundary
```

### GET /at

The ward containing a point.

| Parameter | Required | Meaning |
| --- | --- | --- |
| `lat` | yes | latitude |
| `lng` | yes | longitude |

```sh
curl "localhost:8080/at?lat=-1.2884&lng=36.8233"
```

```json
{
  "ward_code": "1439",
  "name": "NAIROBI CENTRAL",
  "constituency_name": "Starehe",
  "county_code": 47,
  "county_name": "Nairobi County",
  "registered_voters_2022": 52186,
  "centroid": {"lat": -1.28685, "lng": 36.82961}
}
```

Returns 400 if `lat` or `lng` isn't a number, and 404 when no ward boundary covers the point.

## Post offices

### GET /postcodes/{code}

The post offices with a postal code. Most codes have one office.

```sh
curl localhost:8080/postcodes/00100
```

```json
[
  {
    "postal_code": "00100",
    "name": "Nairobi Gpo",
    "county_code": 47,
    "county_source": "geonames",
    "location": {"lat": -1.2863, "lng": 36.8196},
    "location_source": "openstreetmap"
  }
]
```

`location` is null when unknown. `location_source` is `geonames`, `openstreetmap` or empty. `county_source` says how the county was decided: `geonames`, `location`, `manual` or `neighbours`.

### GET /postoffices/near

The closest post offices to a point, nearest first. Offices without a location are skipped.

| Parameter | Required | Meaning |
| --- | --- | --- |
| `lat` | yes | latitude |
| `lng` | yes | longitude |
| `limit` | no | how many to return, 5 by default |

```sh
curl "localhost:8080/postoffices/near?lat=-0.2833&lng=36.0667&limit=2"
```

```json
[
  {"postal_code": "20100", "name": "Nakuru", "county_code": 32, "county_source": "geonames", "location": {"lat": -0.3072, "lng": 36.0722}, "location_source": "geonames", "distance_km": 2.727},
  {"postal_code": "20112", "name": "Lanet", "county_code": 32, "county_source": "geonames", "location": {"lat": -0.3034, "lng": 36.1372}, "location_source": "geonames", "distance_km": 8.152}
]
```

### GET /addresses

Parses a postal address and checks that its box, postal code and town agree.

| Parameter | Required | Meaning |
| --- | --- | --- |
| `q` | yes | the address, such as `P.O. Box 123-00100 Nairobi` |

```sh
curl "localhost:8080/addresses?q=P.O.%20Box%20123-00100%20Nairobi"
```

```json
{
  "box": "123",
  "private_bag": false,
  "postal_code": "00100",
  "town": "Nairobi",
  "post_office": {"postal_code": "00100", "name": "Nairobi Gpo", "county_code": 47, "county_source": "geonames", "location": {"lat": -1.2863, "lng": 36.8196}, "location_source": "openstreetmap"}
}
```

| Status | When |
| --- | --- |
| 200 | the address checks out |
| 400 | there is no P.O. Box number or Private Bag |
| 404 | no post office has that postal code or town |
| 422 | the town and postal code disagree, such as `P.O. Box 90-40100 Eldoret`, where 40100 is Kisumu |

## Search and totals

### GET /search

Fuzzy search across counties, constituencies, wards and post offices, best match first. It forgives typos, so `nakru` finds Nakuru.

| Parameter | Required | Meaning |
| --- | --- | --- |
| `q` | yes | what to look for |
| `limit` | no | how many results, 10 by default |

```sh
curl "localhost:8080/search?q=nakru&limit=3"
```

```json
[
  {"kind": "county", "name": "Nakuru County", "code": "32", "county_code": 32, "score": 0.5},
  {"kind": "post office", "name": "Nakuru", "code": "20100", "county_code": 32, "score": 0.5}
]
```

`kind` is `county`, `constituency`, `ward` or `post office`. `code` is the county code, ward code or postal code, and is left out for constituencies. `score` runs from 0 to 1, where 1 is an exact match.

### GET /voters

Registered voters in 2022 for the whole country.

```sh
curl localhost:8080/voters
```

```json
{"registered_voters_2022": 22096344, "wards": 1450, "complete": false}
```

The total is a little under IEBC's published 22,120,458, which also counts voters registered in the diaspora and in prisons.

### GET /openapi.json

The OpenAPI 3.1 description of every endpoint and the shape of each response.

```sh
curl localhost:8080/openapi.json
```
