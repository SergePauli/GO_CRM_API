package preset

import (
	"fmt"
	"log"
	"strings"

	"github.com/Masterminds/squirrel"
)

// collectColumnsAndJoins собирает колонки и JOIN-ы для пресета
// Возвращает обновленный builder и карту alias -> source
// Используется для построения SQL-запросов с учетом JOIN-ов и алиас
func collectColumnsAndJoins(builder squirrel.SelectBuilder, preset Preset, prefix string) (squirrel.SelectBuilder, map[string]string) {
	aliasToSource := make(map[string]string)
	// JOIN-ы самого пресета
	for _, j := range preset.Joins {
		switch strings.ToUpper(j.Type) {
		case "LEFT JOIN":
			builder = builder.LeftJoin(j.Expr)
		case "RIGHT JOIN":
			builder = builder.RightJoin(j.Expr)
		case "JOIN", "":
			builder = builder.Join(j.Expr)
		default:
			log.Printf("Unknown join type: %s", j.Type)
		}
	}

	for _, f := range preset.Fields {
		switch {
		case f.Type == "computed":
			continue

		case f.Type == "preset" && f.NestedPreset != "":
			nested, err := GetPreset(f.NestedPreset)
			if err != nil {
				log.Printf("Invalid nested preset: %s", f.NestedPreset)
				continue
			}
			nestedPrefix := prefix + f.Alias + "_"

			// Рекурсивный вызов
			var nestedMap map[string]string
			builder, nestedMap = collectColumnsAndJoins(builder, nested, nestedPrefix)

			for k, v := range nestedMap {
				aliasToSource[k] = v
			}

		default:
			alias := f.Alias
			if alias == "" {
				alias = f.Source
			}
			fullAlias := prefix + alias
			expr := f.Source + " AS " + fullAlias
			builder = builder.Column(expr)
			aliasToSource[fullAlias] = f.Source
		}
	}

	

	return builder, aliasToSource
}

// BuildQuery строит SQL-запрос на основе пресета и фильтров
// Использует squirrel для построения запроса с поддержкой JOIN, WHERE и LIMIT
// filters - это карта, где ключи - это имена полей с возможными операциями, например:
// "fieldname__eq": значение - для равенства
// "fieldname__in": []значения - для IN
// "fieldname__lt": значение - для меньше чем
// "fieldname__lte": значение - для меньше или равно
// "fieldname__gt": значение - для больше чем
// "fieldname__gte": значение - для больше или равно

func (p Preset) BuildQuery(filters map[string]interface{}, sorts []string, offset, limit uint64) squirrel.SelectBuilder {
	builder := squirrel.Select().PlaceholderFormat(squirrel.Dollar).From(p.Table)

	// SELECT ...
	var aliasToSource map[string]string
	builder, aliasToSource = collectColumnsAndJoins(builder, p, "")
	

	// WHERE ...
	for rawKey, val := range filters {
		parts := strings.SplitN(rawKey, "__", 2)
		fieldName := parts[0]
		op := "eq"
		if len(parts) == 2 {
			op = parts[1]
		}

		col, ok := aliasToSource[fieldName]
		if !ok {
			log.Printf("⚠️ Unknown filter field: %s", fieldName)
			continue
		}

		switch op {
		case "eq":
			builder = builder.Where(squirrel.Eq{col: val})
		case "in":
			builder = builder.Where(squirrel.Eq{col: val})
		case "lt":
			builder = builder.Where(squirrel.Lt{col: val})
		case "lte":
			builder = builder.Where(squirrel.LtOrEq{col: val})
		case "gt":
			builder = builder.Where(squirrel.Gt{col: val})
		case "gte":
			builder = builder.Where(squirrel.GtOrEq{col: val})
		case "start":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{col: s + "%"})
			}
		case "end":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{col: "%" + s})
			}
		case "cnt":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{col: "%" + s + "%"})
			}
		default:
			log.Printf("⚠️ Unknown filter operation: %s", op)
		}
	}
	// ORDER BY ...
	for _, sort := range sorts {
		parts := strings.Fields(sort) // split by space: "field ASC"
		if len(parts) == 0 {
				continue
		}
		field := parts[0]
		direction := "ASC"
		if len(parts) > 1 {
				dir := strings.ToUpper(parts[1])
				if dir == "DESC" || dir == "ASC" {
						direction = dir
				}
		}

		col, ok := aliasToSource[field]
		if !ok {
				log.Printf("⚠️ Unknown sort field: %s", field)
				continue
		}

		builder = builder.OrderBy(fmt.Sprintf("%s %s", col, direction))
	}
	if limit == 0 {
    limit = 50 // или любое другое дефолтное значение
	}
	return builder.Offset(offset).Limit(limit)
}
