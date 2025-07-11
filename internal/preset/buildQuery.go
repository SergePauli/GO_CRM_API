package preset

import (
	"log"
	"strings"

	"github.com/Masterminds/squirrel"
)

// BuildQuery строит SQL-запрос на основе пресета и фильтров
// Использует squirrel для построения запроса с поддержкой JOIN, WHERE и LIMIT
// filters - это карта, где ключи - это имена полей с возможными операциями, например:
// "fieldname__eq": значение - для равенства
// "fieldname__in": []значения - для IN
// "fieldname__lt": значение - для меньше чем
// "fieldname__lte": значение - для меньше или равно
// "fieldname__gt": значение - для больше чем
// "fieldname__gte": значение - для больше или равно

func (p Preset) BuildQuery(filters map[string]interface{}, offset, limit uint64) squirrel.SelectBuilder {
	builder := squirrel.Select().PlaceholderFormat(squirrel.Dollar).From(p.Table)

	// SELECT ...
	aliasToSource := map[string]string{}

	for _, f := range p.Fields {
		if f.Type == "preset" {
			// Пример: area.card
			parts := strings.Split(f.Source, ".")
			if len(parts) != 2 {
				log.Printf("⚠️ Invalid preset field: %s", f.Source)
				continue
			}
			model, presetName := parts[0], parts[1]
			nestedKey := model + "." + presetName

			nested, err := GetPreset(nestedKey)
			if err != nil {
				log.Printf("⚠️ Nested preset %q not found: %v", nestedKey, err)
				continue
			}

			// добавляем JOIN-ы из вложенного пресета
			for _, j := range nested.Joins {
				switch strings.ToUpper(j.Type) {
				case "LEFT JOIN":
					builder = builder.LeftJoin(j.Expr)
				case "RIGHT JOIN":
					builder = builder.RightJoin(j.Expr)
				case "JOIN", "":
					builder = builder.Join(j.Expr)
				default:
					log.Printf("⚠️ Unknown join type: %s", j.Type)
				}
			}

			// добавляем поля из вложенного пресета
			for _, nf := range nested.Fields {
				colExpr := nf.Source
				alias := nf.Alias
				if alias == "" {
					alias = nf.Source
				}

				nestedAlias := model + "_" + alias // area_name → area_name
				builder = builder.Column(colExpr + " AS " + nestedAlias)
				aliasToSource[nestedAlias] = colExpr
			}
			continue
		}
		if f.Type == "computed" {
			continue // Пропускаем вычисляемые поля, они не нужны в SELECT
		}

		// обычное поле
		expr := f.Source
		if f.Alias != "" && f.Alias != f.Source {
			expr += " AS " + f.Alias
			aliasToSource[f.Alias] = f.Source
		} else {
			aliasToSource[f.Source] = f.Source
		}
		builder = builder.Column(expr)
	}

	// JOIN ...
	for _, j := range p.Joins {
		switch strings.ToUpper(j.Type) {
		case "LEFT JOIN":
			builder = builder.LeftJoin(j.Expr)
		case "RIGHT JOIN":
			builder = builder.RightJoin(j.Expr)
		case "JOIN", "":
			builder = builder.Join(j.Expr)
		default:
			log.Printf("⚠️ Unknown join type: %s", j.Type)
		}
	}

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

	return builder.Offset(offset).Limit(limit)
}
