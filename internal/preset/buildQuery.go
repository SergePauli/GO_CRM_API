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
func collectColumnsAndJoins(builder squirrel.SelectBuilder, preset Preset, prefix string, hasFields map[string]*FieldDef) (squirrel.SelectBuilder, map[string]string) {
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
			builder, nestedMap = collectColumnsAndJoins(builder, nested, nestedPrefix, hasFields)

			for k, v := range nestedMap {
				aliasToSource[k] = v
			}
		case (f.Type == "has_many" || f.Type == "has_one") && f.FKField != "" && f.NestedPreset != "": {
				// Добавим в глобальную карту
				fieldCopy := f // важно: создаем копию, иначе ссылка будет меняться
				// Добавляем алиас для первичного ключа в выборке
				pk := f.PKField
				if pk == "" {
					pk = "id"
				}
				fieldCopy.PKField = prefix + pk
				hasFields[f.Alias] = &fieldCopy
			}
		case f.Source != "":
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

// collectAliasesAndHasMany собирает алиасы и has_many поля из пресета
// Возвращает карту alias -> source и карту alias -> has_many
// Используется для фильтрации и сортировки в запросах
// Рекурсивно обходит вложенные пресеты и собирает алиасы
// Если поле имеет тип "has_many", то добавляет его в aliasToHasMany
// Если поле имеет тип "preset", то рекурсивно обходит вложенный пресет
func collectAliasesAndHasMany(preset Preset) (map[string]string, map[string]bool) {
	aliasToSource := make(map[string]string)
	aliasToHasMany := make(map[string]bool)

	var walk func(p Preset, path []string, inheritedHasMany bool)
	walk = func(p Preset, path []string, inheritedHasMany bool) {
		for _, f := range p.Fields {
			// Полное имя алиаса (например: address_area_name)
			fullAlias := strings.Join(append(path, f.Alias), "_")
			isHasMany := inheritedHasMany || f.Type == "has_many"
			
			switch f.Type {
			case "preset","has_many":
				// Рекурсивно углубляемся в вложенный пресет
				if f.NestedPreset == "" {
					continue
				}
				nested, err := GetPreset(f.NestedPreset)
				if err != nil {
					log.Printf("⚠️ Invalid nested preset: %s", f.NestedPreset)
					continue
				}
				walk(nested, append(path, f.Alias), isHasMany)

			default:
				// Базовые поля: сохраняем alias → SQL и принадлежность к has_many
				if f.Source != ""  {
					aliasToSource[fullAlias] = f.Source
					if isHasMany {
						aliasToHasMany[fullAlias] = true
					}
				}
			}
		}
	}

	walk(preset, []string{}, false)
	return aliasToSource, aliasToHasMany
}
// FiltersTouchHasMany проверяет, есть ли в фильтрах поля, которые относятся к has_many
// Используется для оптимизации запросов: если есть has_many, то добавляем DISTINCT
func SortsTouchHasMany(sorts []string, aliasToHasMany map[string]bool) bool {
	for _, s := range sorts {
		parts := strings.Fields(s)
		if len(parts) > 0 && aliasToHasMany[parts[0]] {
			return true
		}
	}
	return false
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

func (p Preset) BuildQuery(filters map[string]interface{}, sorts []string, offset, limit uint64) (squirrel.SelectBuilder, map[string]*FieldDef) {
	builder := squirrel.Select().PlaceholderFormat(squirrel.Dollar).From(p.Table)
	hasFields := make(map[string]*FieldDef)
	var prefix string = ""
	// SELECT ...
	var aliasToSource map[string]string	
	 builder, aliasToSource = collectColumnsAndJoins(builder, p, prefix, hasFields )
	 log.Printf("aliasToSource = %+v", aliasToSource)
	 
	// Проверяем, есть ли в пресете has_many поля
	_, aliasToHasMany := collectAliasesAndHasMany(p)	
	log.Printf("aliasToHasMany = %+v", aliasToHasMany)
	if FiltersTouchHasMany(filters, aliasToHasMany) || SortsTouchHasMany(sorts, aliasToHasMany) {
		builder = builder.Distinct()
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
			// Если алиас не найден, проверяем, не нужно ли добавить JOIN
			builder = EnsureJoinForMissingField(fieldName, builder, p, "", aliasToSource)
			col, ok = aliasToSource[fieldName]
			if !ok {
				log.Printf("Unknown filter field: %s", fieldName)
				continue
			}
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
	checkedSorts := CheckedSorts(sorts, hasFields)
	log.Printf("Checked sorts: %#v from sorts %#v", checkedSorts, sorts)
	for _, sort := range checkedSorts {
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
				// Если алиас не найден, проверяем, не нужно ли добавить JOIN
			builder = EnsureJoinForMissingField(field, builder, p, "", aliasToSource)
			col, ok = aliasToSource[field]
			if !ok {
				log.Printf("Unknown filter field: %s", field)
				continue
			}
		}

		builder = builder.OrderBy(fmt.Sprintf("%s %s", col, direction))
	}
	if limit == 0 {
    limit = 50 // или любое другое дефолтное значение
	}
	return builder.Offset(offset).Limit(limit), hasFields
}
