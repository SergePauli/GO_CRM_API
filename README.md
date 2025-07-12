# GO_CRM_API - Dynamic API Preset Engine for Go

## 🔧 Overview

This project implements a **universal, high-performance API engine** in Go that dynamically builds SQL queries and JSON responses using declarative **presets**. It's compatible with frontends expecting Ransack-style filters and supports deep nested associations and computed fields.


📌 Go API for CRM-like backend — experimental rewrite of Rails monolith using `pgx`, `squirrel`, and clean architecture principles.

---
## 🚀 Features

- One universal API endpoint (`POST /api/index`) for all models
- Declarative model presets with:
  - Fields (basic, nested, computed)
  - SQL Joins
  - JSON formatting logic
- Ransack-style filters with `__eq`, `__cnt`, `__in`, etc.
- Recursive support for nested presets (e.g. `contragent_address.address.area`)
- High-performance JSON output via pgx.Rows scanning
- Accurate `COUNT(*)` via `POST /api/count`
- Dynamic, maintainable, and frontend-friendly

---


## 🔁 Current Branch: `pgx-squirrel`

> ✅ Experimental branch using `pgx` + `squirrel`  
> 🔄 Designed for programmatic SQL and composable filters/presets  
> 📚 Similar to Rails' ActiveRecord + Ransack + JSON presets

---

## 📦 Tech stack

- Go 1.24+
- pgx (PostgreSQL native driver)
- squirrel (SQL builder)
- GORM (in other branches)
- Redis (caching layer)
- Gin or Chi (routing — to be added)

---

## 🛠 Dev setup

```bash
go mod tidy
cp .env.example .env
make run     # or make watch
```
---


## 🗂 Project Structure

```
GO_CRM_API/
├── cmd/                   # Entry point (main.go)
├── internal/
│   ├── config/            # Loads .env and app config
│   ├── db/                # Init Redis, Postgres, shared DB handle
│   ├── handler/           # HTTP handlers: index, count
│   ├── preset/            # Preset registry, SQL builders, JSON scan logic
│   ├── router/            # Routes setup
│   └── model/             # (Optional) Go structs if needed for migrations/typing
├── Makefile
├── go.mod
├── .env
└── README.md
```

---


## 📦 Preset Structure

```go
type FieldDef struct {
	Source       string // SQL column or expression
	Alias        string // JSON key (optional)
	Type         string // "int", "string", "preset", "computed"
	Formatter    func(any) any // Optional computed formatter
	NestedPreset string // For type = "preset"
}

type JoinSpec struct {
	Type string // e.g. "LEFT JOIN"
	Expr string // e.g. "addresses ON addresses.id = users.address_id"
}

type Preset struct {
	Table  string
	Fields []FieldDef
	Joins  []JoinSpec
}
```

Registered via:

```go
func InitAllPresets() {
	RegisterAddressPresets()
	RegisterContragentAddressPresets()
	...
}
```

Example preset:

```go
Registry["contragent_address.edit"] = Preset{
	Table: "contragent_addresses",
	Fields: []FieldDef{
		{Source: "contragent_addresses.id", Alias: "id", Type: "int"},
		{Source: "contragent_addresses.kind", Alias: "kind", Type: "string"},
		{Source: "address", Type: "preset", NestedPreset: "address.edit"},
	},
	Joins: []JoinSpec{
		{Type: "LEFT JOIN", Expr: "addresses ON addresses.id = contragent_addresses.address_id"},
	},
}
```

---

## 📡 Endpoints

### `POST /api/index`

Returns data for a model using a given preset.

#### Request Body:

```json
{
  "model": "contragent_address",
  "preset": "edit",
  "offset": 0,
  "limit": 10,
  "filters": {
    "kind__eq": 0,
    "address_value__cnt": "ул.",
    "address_area_name__start": "Респ"
  }
}
```

#### Sample Response:

```json
[
  {
    "id": 1,
    "kind": 0,
    "address": {
      "id": 139,
      "value": "г. Улан-Удэ...",
      "area": {
        "id": 3,
        "name": "Республика Бурятия"
      }
    }
  }, ...
]
```

### `POST /api/count`

Returns total count of records with the same filters.

#### Request Body:

```json
{
  "model": "contragent_address",
  "preset": "edit",
  "filters": {
    "address_area_name__start": "Респ"
  }
}
```

#### Response:

```json
{
  "count": 112
}
```

---

## 🔍 Supported Filters

| Operator  | Description          |
| --------- | -------------------- |
| `__eq`    | equals               |
| `__in`    | list of values       |
| `__lt`    | less than            |
| `__lte`   | less than or equal   |
| `__gt`    | greater than         |
| `__gte`   | greater or equal     |
| `__start` | starts with (`LIKE`) |
| `__end`   | ends with (`LIKE`)   |
| `__cnt`   | contains (`LIKE`)    |

---

## ⏱ Performance

All queries are built using `github.com/Masterminds/squirrel` and executed with `pgx`.

- Average query time: **< 10 ms**
- Cached count queries are even faster on repetition
- Recursion is lazy and includes only needed joins

---

## 📌 Roadmap

-

---

## ✅ Example cURL

```bash
curl -X POST http://localhost:8080/api/index \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "contragent_address",
    "preset": "edit",
    "offset": 0,
    "limit": 10,
    "filters": {
      "kind__eq": 0,
      "address_value__cnt": "ул.",
      "address_area_name__start": "Респ"
    }
  }'
```

---

## 🧠 Credits

Author: [@SergePauli](https://github.com/SergePauli)

With inspiration from Rails' `Ransack`, Go's simplicity, and the power of structured SQL builders.

---

*Last updated: 2025-07-12*

