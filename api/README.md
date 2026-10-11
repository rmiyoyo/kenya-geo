# HTTP API

The `api` package serves Kenya's counties, constituencies, wards and post offices as JSON, with 2019 census figures for every county: population by sex, urban and rural, households and living conditions, and religious affiliation, and the same for Kenya as a whole. Run it with `kenyageo-server`, or mount `api.New` in your own server.

This page describes version 0.3.1. [CHANGELOG.md](https://github.com/rmiyoyo/kenya-geo/blob/main/CHANGELOG.md) lists what changed in each version.

Every endpoint is a `GET`. Responses allow requests from any origin, so a web page can call the API directly. A machine-readable [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0) description is served at `/openapi.json`.

## Conventions

### Base URL

The paths below are relative to wherever the API is mounted. `kenyageo-server` serves them from the root, so `GET /counties` is `http://localhost:8080/counties`. If you mount the handler under a prefix, such as `/geo/`, add the prefix: `/geo/counties`.

### Errors

A request that fails returns a JSON object with an `error` message:

```json
{"error": "no county \"99\""}
```

| Status | When |
| --- | --- |
| `400` | a query parameter is missing or not a number, or `limit` is outside 1 to 100 |
| `404` | nothing matches: no such county, ward, constituency or postal code, or no ward at a point |
| `422` | an address's town and postal code disagree |

### Identifying places

- **Counties** take their code (`32`) or their name, with or without "County" and in any case (`nakuru`, `Nakuru County`).
- **Constituencies** take their name in any case (`kibra`, `Nakuru Town East`).
- **Wards** take their four-digit IEBC code (`0040`).
- **Postal codes** are five digits (`00100`). Several post offices can share one code.

### Coordinates

Points are WGS 84 latitude and longitude in decimal degrees, passed as `lat` and `lng`. Locations in responses use the same names: `{"lat": -1.2863, "lng": 36.8196}`. Bounding boxes are `[west, south, east, north]`.

### Limits

Endpoints that take `limit` accept 1 to 100.

## Counties

### GET /counties

All 47 counties, in code order.

```http
GET /counties
```

Each county looks like the one from `GET /counties/{county}`.

### GET /counties/{county}

One county, by code or name.

```http
GET /counties/32
```

```json
{
  "code": 32,
  "name": "Nakuru County",
  "iso_code": "KE-31",
  "former_province": "Rift Valley",
  "headquarters": "Nakuru",
  "population": 2162202,
  "population_male": 1077272,
  "population_female": 1084835,
  "population_intersex": 95,
  "population_urban": 1047080,
  "population_rural": 1115122,
  "area_km2": 7462.4,
  "centroid": {"lat": -0.46402, "lng": 36.07634},
  "bbox": [35.41247, -1.15591, 36.59541, 0.23466],
  "neighbours": [18, 22, 30, 31, 33, 34, 35],
  "living": {
    "households": 616046,
    "average_household_size": 3.5,
    "electricity_pct": 64.4,
    "piped_water_pct": 27.5,
    "internet_pct": 26.8,
    "mobile_phone_pct": 52.7,
    "school_attendance_pct": 96.4
  },
  "religion": {
    "total": 2142667,
    "catholic": 349527,
    "protestant": 703881,
    "evangelical": 647780,
    "african_instituted": 154007,
    "orthodox": 12182,
    "other_christian": 140000,
    "islam": 25479,
    "hindu": 1660,
    "traditionist": 4568,
    "other_religion": 30739,
    "no_religion": 67640,
    "dont_know": 4937,
    "not_stated": 267
  }
}
```

| Field | Meaning |
| --- | --- |
| `code` | county code, 1 to 47 |
| `iso_code` | ISO 3166-2 code |
| `population`, `population_male`, `population_female`, `population_intersex` | 2019 census (KNBS) |
| `population_urban`, `population_rural` | people living in urban and rural areas, from 2019 census Volume II; they add up to `population` |
| `area_km2` | land area in square kilometres |
| `centroid`, `bbox` | from the simplified boundary |
| `neighbours` | codes of counties that share a border |
| `living` | living conditions from the 2019 census, below |
| `religion` | people by religious affiliation from the 2019 census, below |

`living` comes from the volumes of the [2019 Kenya Population and Housing Census](https://open.africa/dataset/2019-kenya-population-and-housing-census) published by KNBS. Percentages are out of 100.

| Field | Meaning | Census table |
| --- | --- | --- |
| `households` | households, conventional and group quarters | Volume I: population, households and average household size by county |
| `average_household_size` | people living in households divided by households, to one decimal place | Volume I, as above |
| `electricity_pct` | conventional households lit mainly by mains electricity | Volume IV: main type of lighting fuel |
| `piped_water_pct` | conventional households whose main drinking water is piped into the dwelling or to the yard or plot | Volume IV: main source of drinking water |
| `internet_pct` | people aged 3 and over who used the internet | Volume IV: population aged 3 and above using the internet |
| `mobile_phone_pct` | people aged 3 and over who own a mobile phone | Volume IV: population aged 3 and above owning a mobile phone |
| `school_attendance_pct` | children aged 6 to 17 at school or another learning institution | Volume IV: school attendance status by special age groups, adding the 6–13 and 14–17 groups |

`religion` counts people in households by religious affiliation, from Volume IV's distribution of population by religious affiliation and county. `total` is the population in households, so it is a little smaller than `population`, and the other fields add up to it. The Christian groups are `catholic`, `protestant`, `evangelical`, `african_instituted`, `orthodox` and `other_christian`; the rest are `islam`, `hindu`, `traditionist`, `other_religion`, `no_religion`, `dont_know` and `not_stated`.

### GET /counties/{county}/constituencies

The county's constituency names, sorted.

```http
GET /counties/32/constituencies
```

```json
["Bahati", "Gilgil", "Kuresoi North", "Kuresoi South", "Molo", "Naivasha", "Nakuru Town East", "Nakuru Town West", "Njoro", "Rongai", "Subukia"]
```

### GET /counties/{county}/neighbours

The counties that share a border with this one, as full county objects.

```http
GET /counties/32/neighbours
```

### GET /counties/{county}/wards

Every ward in the county. Each ward looks like the one from `GET /wards/{code}`.

```http
GET /counties/32/wards
```

### GET /counties/{county}/voters

Registered voters for the 2022 election, summed over the county's wards.

```http
GET /counties/32/voters
```

```json
{"registered_voters_2022": 1054856, "wards": 55, "complete": true}
```

`complete` is false when one or more wards have no published figure, so the total is a lower bound.

### GET /counties/{county}/postoffices

Post offices in the county. The list is empty, not an error, for a county with none.

```http
GET /counties/32/postoffices
```

Each post office looks like the ones from `GET /postcodes/{code}`.

### GET /counties/{county}/boundary

The county outline as a GeoJSON Feature, served as `application/geo+json`. The outline is simplified, so treat points near a border as approximate.

```http
GET /counties/32/boundary
```

### GET /counties/at

The county containing a point.

```http
GET /counties/at?lat=-0.0917&lng=34.768
```

Returns a county, or `404` when the point is outside every county.

## Constituencies

### GET /constituencies/{name}/wards

The wards in a constituency.

```http
GET /constituencies/kibra/wards
```

### GET /constituencies/{name}/voters

Registered voters in a constituency, in the same shape as the county total.

```http
GET /constituencies/kibra/voters
```

```json
{"registered_voters_2022": 128282, "wards": 5, "complete": true}
```

## Wards

### GET /wards/{code}

One ward, by its IEBC code.

```http
GET /wards/0040
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

Ward names are in capitals, as the IEBC publishes them. `registered_voters_2022` is null when there is no published figure, and `centroid` is null when there is no boundary.

### GET /wards/{code}/boundary

The ward outline as a GeoJSON Feature, served as `application/geo+json`. Returns `404` for the few wards with no known boundary.

```http
GET /wards/0040/boundary
```

### GET /at

The ward containing a point.

```http
GET /at?lat=-1.2884&lng=36.8233
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

Returns `404` when no ward boundary contains the point.

## Voters

### GET /voters

The national total of registered voters for 2022.

```http
GET /voters
```

```json
{"registered_voters_2022": 22096344, "wards": 1450, "complete": false}
```

## Kenya

### GET /national

Figures for Kenya as a whole: the 2019 census population, urban and rural population, living conditions and religion, and the 2022 registered voter total.

```http
GET /national
```

```json
{
  "population": 47564296,
  "population_male": 23548056,
  "population_female": 24014716,
  "population_intersex": 1524,
  "area_km2": 580876.5,
  "population_urban": 14831700,
  "population_rural": 32732596,
  "living": {
    "households": 12143913,
    "average_household_size": 3.9,
    "electricity_pct": 50.4,
    "piped_water_pct": 24.2,
    "internet_pct": 22.6,
    "mobile_phone_pct": 47.3,
    "school_attendance_pct": 87.8
  },
  "religion": {
    "total": 47213282,
    "catholic": 9726169,
    "protestant": 15777473,
    "evangelical": 9648690,
    "african_instituted": 3292573,
    "orthodox": 201263,
    "other_christian": 1732911,
    "islam": 5152194,
    "hindu": 60287,
    "traditionist": 318727,
    "other_religion": 467083,
    "no_religion": 755750,
    "dont_know": 73253,
    "not_stated": 6909
  },
  "voters": {"registered_voters_2022": 22096344, "wards": 1450, "complete": false}
}
```

`population`, its parts by sex and `area_km2` add up the 47 counties. `living` and `religion` have the same fields as on a county; Kenya's figures come from the census's national rows, except school attendance, which is worked out from the county totals because that table has no national row. `voters` is the same as [`GET /voters`](#get-voters).

## Post offices

### GET /postcodes/{code}

The post offices with a postal code.

```http
GET /postcodes/00100
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

`location` is null when no source places the office precisely, and `location_source` is then empty; otherwise it is `geonames` or `openstreetmap`. `county_source` says how the county was worked out:

| `county_source` | Meaning |
| --- | --- |
| `geonames` | the county GeoNames gives |
| `location` | the county the office's location falls in |
| `manual` | set by hand |
| `neighbours` | the county most offices with nearby postal codes are in |

### GET /postoffices/near

The post offices closest to a point, nearest first. Offices without a location are skipped.

| Parameter | |
| --- | --- |
| `lat`, `lng` | the point |
| `limit` | how many to return, default 5 |

```http
GET /postoffices/near?lat=-0.2833&lng=36.0667&limit=2
```

```json
[
  {"postal_code": "20100", "name": "Nakuru", "county_code": 32, "county_source": "geonames", "location": {"lat": -0.3072, "lng": 36.0722}, "location_source": "geonames", "distance_km": 2.727},
  {"postal_code": "20112", "name": "Lanet", "county_code": 32, "county_source": "geonames", "location": {"lat": -0.3034, "lng": 36.1372}, "location_source": "geonames", "distance_km": 8.152}
]
```

`distance_km` is the great-circle distance.

### GET /addresses

Parses a Kenyan postal address and checks it against the post office list.

| Parameter | |
| --- | --- |
| `q` | the address, such as `P.O. Box 123-00100 Nairobi` |

```http
GET /addresses?q=P.O.%20Box%20123-00100%20Nairobi
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

The parser accepts `P.O. Box`, `Box` and `Private Bag`, with the postal code joined to the box by a dash or written separately. It fails with:

| Status | When |
| --- | --- |
| `400` | there is no box number or private bag |
| `404` | no post office has that postal code or town |
| `422` | the town and the postal code belong to different post offices |

## Search

### GET /search

Fuzzy search across counties, constituencies, wards and post offices, so `nakru` still finds Nakuru.

| Parameter | |
| --- | --- |
| `q` | what to look for, required |
| `limit` | how many results, default 10 |

```http
GET /search?q=nakru&limit=3
```

```json
[
  {"kind": "county", "name": "Nakuru County", "code": "32", "county_code": 32, "score": 0.5},
  {"kind": "post office", "name": "Nakuru", "code": "20100", "county_code": 32, "score": 0.5}
]
```

`kind` is `county`, `constituency`, `ward` or `post office`. `code` is the county code, ward code or postal code, and is left out for constituencies. Results are sorted best first; `score` runs from 0 to 1, where 1 is an exact match.

## OpenAPI

### GET /openapi.json

The OpenAPI 3.1 description of every endpoint here, with the shape of each response. Load it into Swagger UI, Postman or a client generator such as `oapi-codegen`.

```http
GET /openapi.json
```
