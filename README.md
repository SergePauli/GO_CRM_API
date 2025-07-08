# GO_CRM_API

📌 Go API for CRM-like backend — experimental rewrite of Rails monolith using `pgx`, `squirrel`, and clean architecture principles.

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
