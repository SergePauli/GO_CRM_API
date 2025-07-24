package preset

import (
	"fmt"
	"log"
	"strings"
)

// toNestedJSON преобразует плоские данные в структуру JSON с учетом вложенных пресетов и has_many отношений
// flat - это плоский словарь, где ключи - это полные имена полей
// hasManyData - это данные для полей типа "has_many", сгруппированные по алиасам
func toNestedJSON(p Preset, flat map[string]any, hasManyData map[string]map[string][]map[string]any) map[string]any {
	result := make(map[string]any)

	for _, field := range p.Fields {
		alias := field.Alias
		if alias == "" {
			alias = field.Source
		}
		fullKey := p.Table + "_" + alias

		switch field.Type {
		case "preset":
			if field.NestedPreset != "" {
				// Рекурсивный вызов вложенного пресета
				nestedPreset, err := GetPreset(field.NestedPreset)
				if err != nil {
					log.Printf("toNestedJSON: preset '%s' not found", field.NestedPreset)
					continue
				}

				// Собираем flat-префикс для вложенного пресета
				prefix := alias + "_"
				subFlat := make(map[string]any)
				for k, v := range flat {
					if strings.HasPrefix(k, prefix) {
						subFlat[strings.TrimPrefix(k, prefix)] = v
					}
				}

				nested := toNestedJSON(nestedPreset, subFlat, hasManyData)
				result[alias] = nested
			}

		case "has_many":
			if hasManyData != nil {
				// Получаем PK текущей записи (всегда string)
				pkVal := fmt.Sprintf("%v", flat[field.PKField])
				grouped, ok := hasManyData[alias]
				if ok {
					result[alias] = grouped[pkVal]
				} else {
					result[alias] = []any{} // пустой массив
				}
			}

		case "computed":
			// игнорируем, вычисляется отдельно
			continue

		default:
			// Прямое копирование значения
			if val, ok := flat[fullKey]; ok {
				result[alias] = val
			}
		}
	}

	return result
}
// DataToJSON converts cached rows and hasMany data into a JSON-like structure
// This function is used to build the final JSON response for the resolver.
func (p Preset) DataToJSON(
	cachedRows []map[string]any,
	hasManyData map[string]map[string][]map[string]any,
) ([]any, error) {
	var results []any
	

	for _, flat := range cachedRows {
		nested := toNestedJSON(p, flat, hasManyData)
		results = append(results, nested)
	}

	return results, nil
}