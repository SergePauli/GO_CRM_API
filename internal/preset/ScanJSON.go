package preset

import (
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
)

// collectColumns собирает все колонки для SQL-запроса из пресета
// с учетом вложенных пресетов и алиасов
// prefix - префикс для алиасов, например "area_"
func collectColumns(p Preset, prefix string) []string {
	var columns []string

	for _, f := range p.Fields {
		if f.Type == "computed" {
			continue // не добавляем вычисляемые поля в SELECT
		}

		if f.NestedPreset != "" {
			nested, err := GetPreset(f.NestedPreset)
			if err != nil {
				continue
			}
			newPrefix := prefix + f.Alias + "_"
			columns = append(columns, collectColumns(nested, newPrefix)...)
		} else {
			alias := f.Alias
			if alias == "" {
				alias = f.Source
			}
			columns = append(columns, prefix+alias)
		}
	}

	return columns
}

// ScanRowsToNestedMap сканирует строки и строит вложенные JSON-объекты
// Использует alias'ы из пресета, включая вложенные preset и has_many
func ScanRowsToNestedMap(rows pgx.Rows, p Preset, hasManyData map[string]map[int64][]map[string]any) ([]map[string]any, error) {
	result := []map[string]any{}

	cols := rows.FieldDescriptions()
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		// плоская строка: map[alias]value
		flat := make(map[string]any)
		for i, col := range cols {
			flat[string(col.Name)] = values[i]
		}

		// вложенная структура по пресету
		nested := buildNestedJSON(p, flat, hasManyData)
		result = append(result, nested)
	}

	return result, nil
}


// buildNestedJSON строит вложенную структуру JSON из плоской карты
// flat - плоская карта с данными
// p - пресет, определяющий структуру вложенности
// prefix - префикс для ключей, например "area_"
// Возвращает вложенную карту с данными
// buildNestedJSON строит вложенный JSON-объект по плоской строке и пресету
func buildNestedJSON(p Preset, flat map[string]any, hasManyData map[string]map[int64][]map[string]any) map[string]any {
	result := make(map[string]any)	
	for _, f := range p.Fields {
		switch {
		case f.Type == "computed":
			continue

		case f.Type == "preset" && f.NestedPreset != "":
			nestedPreset, err := GetPreset(f.NestedPreset)
			if err != nil {
				log.Printf("❌ Invalid nested preset in buildNestedJSON: %s", f.NestedPreset)
				continue
			}

			nestedPrefix := f.Alias + "_"
			subFlat := make(map[string]any)

			// Сначала добавляем все поля из nestedPreset.Fields
			knownKeys := make(map[string]struct{})
			for _, nf := range nestedPreset.Fields {
				key := nestedPrefix + nf.Alias
				val := flat[key]
				if val != nil {
					subFlat[nf.Alias] = val
				}
				knownKeys[key] = struct{}{}
			}

			// Теперь ищем всё в flat, что начинается с nestedPrefix,
			// но не входит в nestedPreset.Fields — и добавляем это без префикса
			for k, v := range flat {
				if !strings.HasPrefix(k, nestedPrefix) {
					continue
				}
				if _, ok := knownKeys[k]; ok {
				continue // уже добавлено
			}
			// Сохраняем без префикса
			unprefixed := strings.TrimPrefix(k, nestedPrefix)
			subFlat[unprefixed] = v
			}

			//log.Printf("subFlat: %#v in preset %v flat %#v", subFlat, f.Alias, flat)
			result[f.Alias] = buildNestedJSON(nestedPreset, subFlat, nil)

		case f.Type == "has_many" && f.FKField != "":
			// Предполагается, что PK родителя — это "id"
			idVal, ok := flat["id"].(int64)
			if !ok {
				result[f.Alias] = []any{}
				continue
			}
			
			result[f.Alias] = []any{}
			if hasManyData != nil {
				if group, ok := hasManyData[f.Alias]; ok {
					if items, ok := group[idVal]; ok {
					if f.NestedPreset != "" {
						nestedPreset, err := GetPreset(f.NestedPreset)
						if err != nil {
						log.Printf("Failed to load nested preset %s: %v", f.NestedPreset, err)
						continue
					}
				
				
				// Применяем рекурсивное преобразование
				nested, err := nestedPreset.ScanJSON(nil, items, hasManyData)
				if err != nil {
						log.Printf("Failed to scan nested has_many data: %v", err)
						continue
				}
					result[f.Alias] = nested
				//log.Printf("Inserting nested %#v items into %s (parentID=%d)", nested, f.Alias, idVal)
			} else {
				// Фолбэк — просто отдаем без вложенной обработки
				result[f.Alias] = items
				log.Printf("Inserting %d flat items into %s (parentID=%d)", len(items), f.Alias, idVal)
			}
			continue
		}
	}
}
		default:
			val := flat[f.Alias]
			if val == nil {
				result[f.Alias] = nil
			} else {
				result[f.Alias] = val
			}
		}
	}
	
	return result
}



// ScanJSON выполняет сканирование результатов запроса в формате JSON
func (p Preset) ScanJSON( rows pgx.Rows, cache []map[string]any, hasManyData map[string]map[int64][]map[string]any) ([]any, error) {
	var results []any
	var records []map[string]any
	columns := collectColumns(p, "")
	//log.Printf("columns: %#v", columns)
	// Используем кэш если есть, иначе сканируем rows
  
	if cache != nil {
		// Если кэш содержит только одну запись, то flatten-им
		if (len(cache) == 1) {
			flat := make(map[string]any)
			for _, fieldName := range columns {
				flat[fieldName] = cache[0][p.Table + "_" + fieldName]				
			}
			records = append(records, flat)
		} else {
			// Если кэш содержит несколько записей, то оставляем как есть
			records = cache
		}	
	} else {		
		for rows.Next() {
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range ptrs {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return nil, err
			}

			flat := make(map[string]any)
			for i, name := range columns {
				flat[name] = values[i]				
			}
			records = append(records, flat)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	
	// Преобразуем flat → nested
	for _, flat := range records {		
		nested := buildNestedJSON(p, flat, hasManyData)
		results = append(results, nested)
		
	}

		
	return results, nil
}
