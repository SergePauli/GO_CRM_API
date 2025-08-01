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
func collectColumnsAndJoins(builder squirrel.SelectBuilder, preset Preset, prefix string, hasFields map[string]*FieldDef, whereFields *[]FieldDef,) (squirrel.SelectBuilder, map[string]string) {
	aliasToSource := make(map[string]string)

	for _, f := range preset.Fields {
		if (f.Type == "has_many" || f.Type == "has_one") && f.Where != "" {
			fieldCopy := f
			*whereFields = append(*whereFields, fieldCopy)			
		} else if (f.Where == "" && f.Type == "has_one") {
			log.Printf("⚠️ has_one поле %s не содержит ограничения в Where — может вернуть несколько строк", f.Alias)
		}	
		switch {
		case f.Type == "computed":
			continue

		case (f.Type == "belongs_to" || f.Type == "has_one") && f.NestedPreset != "":
			nested, err := GetPreset(f.NestedPreset)
			if err != nil {
				log.Printf("Invalid nested preset: %s", f.NestedPreset)
				continue
			}
			// Получаем source и alias
			parentAlias := preset.Table // например, "contragents"
			if prefix != "" {
				parentAlias =  strings.TrimRight(prefix,"_") // убираем последний "_"
			}			
			childAlias := prefix + f.Alias // например, "real_address"
			childTable := nested.Table // например, "addresses"
			pk := f.PKField
				if pk == "" {
					pk = "id"
			}
			fk := f.FKField
				if fk == "" {
				fk = f.Alias + "_id" 
			}
			var onClause string
			if f.Type == "has_one" {
				onClause = fmt.Sprintf("%s.%s = %s.%s", childAlias, fk, parentAlias, pk)
			} else { // f.Type == preset
				onClause = fmt.Sprintf("%s.%s = %s.%s", parentAlias, fk, childAlias, pk)
			}	

			// Добавляем JOIN
			builder = builder.LeftJoin(fmt.Sprintf("%s AS %s ON %s", childTable, childAlias, onClause))
			aliasToSource[childAlias] = childTable

			log.Printf("✅ Добавлен JOIN для  %s: %s", f.Alias, onClause)
			nestedPrefix := childAlias + "_"

			// Рекурсивный вызов
			var nestedMap map[string]string
			builder, nestedMap = collectColumnsAndJoins(builder, nested, nestedPrefix, hasFields, whereFields)

			for k, v := range nestedMap {
				aliasToSource[k] = v
			}
		case (f.Type == "has_many") && f.FKField != "" && f.NestedPreset != "": {
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
			var fieldAlias string
			if prefix == "" {
    		fieldAlias = f.Source
			} else {
    		fieldAlias = strings.Replace(fullAlias, "_"+alias, "."+alias,1) // заменяем на SQL-валидный путь
			}
			expr := fieldAlias + " AS " + fullAlias
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
// Если поле имеет тип "belongs_to", то рекурсивно обходит вложенный пресет
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
			case "belongs_to","has_many","has_one":
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
	// Создаём whereFields (как список значений, не ссылок)
	var whereFields []FieldDef
	 builder, aliasToSource = collectColumnsAndJoins(builder, p, prefix, hasFields, &whereFields )
	 log.Printf("aliasToSource = %+v whereFields = %+v", aliasToSource, whereFields)
	 
	// Проверяем, есть ли в пресете has_many поля
	_, aliasToHasMany := collectAliasesAndHasMany(p)	
	log.Printf("aliasToHasMany = %+v", aliasToHasMany)
	if FiltersTouchHasMany(filters, aliasToHasMany) || SortsTouchHasMany(sorts, aliasToHasMany) {
		builder = builder.Distinct()
	}

	// WHERE ...
	for _, f := range whereFields  {
		
		if f.Where == "" && !(f.Type == "has_one" || f.Type == "has_many") {
			continue
		}

		// добавить через squirrel.Expr
		builder = builder.Where(squirrel.Expr(f.Where))	
	}
	
	for rawKey, val := range filters {
		parts := strings.SplitN(rawKey, "__", 2)
		
		fieldName := parts[0] 
		
		op := "eq"
		if len(parts) == 2 {
			op = parts[1]
		}
		_, ok := aliasToSource[fieldName]
		if !ok {
			// Если алиас не найден, проверяем, не нужно ли добавить JOIN
			builder = EnsureJoinForMissingField(fieldName, builder, p, "", aliasToSource)
			_, ok = aliasToSource[fieldName]
			if !ok {
				log.Printf("Unknown filter field: %s", fieldName)
				continue
			}
		} 
		switch op {
		case "eq":
			builder = builder.Where(squirrel.Eq{fieldName: val})
		case "in":
			builder = builder.Where(squirrel.Eq{fieldName: val})
		case "lt":
			builder = builder.Where(squirrel.Lt{fieldName: val})
		case "lte":
			builder = builder.Where(squirrel.LtOrEq{fieldName: val})
		case "gt":
			builder = builder.Where(squirrel.Gt{fieldName: val})
		case "gte":
			builder = builder.Where(squirrel.GtOrEq{fieldName: val})
		case "start":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{fieldName: s + "%"})
			}
		case "end":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{fieldName: "%" + s})
			}
		case "cnt":
			if s, ok := val.(string); ok {
				builder = builder.Where(squirrel.Like{fieldName: "%" + s + "%"})
			}
		default:
			log.Printf("⚠️ Unknown filter operation: %s", op)
		}
	}
	
	// ORDER BY ...
	checkedSorts := CheckedSorts(sorts, hasFields)
	//log.Printf("Checked sorts: %#v from sorts %#v", checkedSorts, sorts)
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

		_, ok := aliasToSource[field]
		if !ok {
				// Если алиас не найден, проверяем, не нужно ли добавить JOIN
			builder = EnsureJoinForMissingField(field, builder, p, "", aliasToSource)
			_, ok = aliasToSource[field]
			if !ok {
				log.Printf("Unknown filter field: %s", field)
				continue
			}
		}
		builder = builder.OrderBy(fmt.Sprintf("%s %s", field, direction))
	}
	if limit == 0 {
    limit = 50 // или любое другое дефолтное значение
	}
	return builder.Offset(offset).Limit(limit), hasFields
}
