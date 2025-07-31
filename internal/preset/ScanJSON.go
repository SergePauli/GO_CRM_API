package preset

import (
	"fmt"

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

func (p Preset) ParseFlatRow(flat map[string]any) (map[string]any, error) {
	nested := make(map[string]any)

	for _, field := range p.Fields {
		switch field.Type {
		case "computed":
			if field.Formatter != nil {
				nested[field.Alias] = field.Formatter(flat)
			}
		case "belongs_to":
			nestedPreset, ok := Registry[field.NestedPreset]
			if !ok {
				return nil, fmt.Errorf("nested preset '%s' not found", field.NestedPreset)
			}
			subFlat := extractSubMapByPreset(flat, field.Alias, nestedPreset)			
			subNested, err := nestedPreset.ParseFlatRow(subFlat)
			if err != nil {
				return nil, err
			}
			nested[field.Alias] = subNested
		default:
			if field.Internal {
				continue // пропускаем внутренние поля
			} else {
			// Прямое копирование значения
			nested[field.Alias] = flat[field.Alias]
			}
		}
	}

	return nested, nil
}


// extractSubMap извлекает подсловарь из flat по префиксу
func extractSubMapByPreset(flat map[string]any, parentAlias string, nested Preset) map[string]any {
	subMap := make(map[string]any)  
	for _, f := range nested.Fields {
		if f.Type == "computed" {
			continue
		}
		// Вложенный пресет: рекурсивно извлекаем под-словарь
		if f.Type == "belongs_to" && f.NestedPreset != "" {
			nestedPreset := Registry[f.NestedPreset]
			nestedPrefix := parentAlias+"_"+f.Alias
			nestedMap := extractSubMapByPreset(flat, nestedPrefix, nestedPreset)			
			// Просто переносим расплющенные поля в текущий subMap
			for k, v := range nestedMap {
				subMap[f.Alias+"_"+k] = v
			}			
			continue
		}
		alias := f.Alias
		if alias == "" {
			alias = f.Source
		}

		key := parentAlias + "_" + alias		
		if val, ok := flat[key]; ok {
			subMap[alias] = val
		}
	}

	return subMap
}




// ParseRowsToJSON преобразует строки из базы данных в JSON-формат
// Использует collectColumns для получения списка колонок
// Возвращает массив JSON-объектов, соответствующих пресету
// rows - это результат выполнения SQL-запроса
// Преобразует каждую строку в map[string]any с учетом алиасов
// и вложенных пресетов
func (p Preset) ParseRowsToJSON(rows pgx.Rows) ([]map[string]any, error) {
	var results []map[string]any
	defer rows.Close()

	// Собираем список всех колонок, которые должны прийти из запроса
	columns := collectColumns(p, "")

	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range ptrs {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		flat := make(map[string]any)
		for i, name := range columns {
			flat[name] = values[i]
		}

		// Преобразуем flat → nested по типам полей
		nested := make(map[string]any)
		for _, field := range p.Fields {
			switch field.Type {
			case "computed":
				if field.Formatter != nil {
					nested[field.Alias] = field.Formatter(flat)
				}
			case "belongs_to","has_one":
				nestedPreset, ok := Registry[field.NestedPreset]
				if !ok {
					return nil, fmt.Errorf("preset '%s': nested preset '%s' not found", p.Table, field.NestedPreset)
				}				
				subFlat := extractSubMapByPreset(flat, field.Alias, nestedPreset)							
				subNested, err := nestedPreset.ParseFlatRow(subFlat)
				if err != nil {
					return nil, fmt.Errorf("nested preset '%s': %w", field.NestedPreset, err)
				}
				nested[field.Alias] = subNested
			default:
				// Простое значение по алиасу				
				nested[field.Alias] = flat[field.Alias]				
			}
		}

		results = append(results, nested)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
