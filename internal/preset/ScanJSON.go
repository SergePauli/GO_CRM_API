package preset

import (
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

// buildNestedJSON строит вложенную структуру JSON из плоской карты
// flat - плоская карта с данными
// p - пресет, определяющий структуру вложенности
// prefix - префикс для ключей, например "area_"
// Возвращает вложенную карту с данными
func buildNestedJSON(flat map[string]any, p Preset, prefix string) map[string]any {
	result := make(map[string]any)

	for _, f := range p.Fields {
		if f.NestedPreset != "" {
			nested, err := GetPreset(f.NestedPreset)
			if err != nil {
				continue
			}
			newPrefix := prefix + f.Alias + "_"

			if f.Type == "array" {
				// пока нет поддержки группировки массива — можно реализовать позже
				result[f.Alias] = []any{} // заглушка
			} else {
				result[f.Alias] = buildNestedJSON(flat, nested, newPrefix)
			}
		} else if f.Type == "computed" {
			if f.Formatter != nil {
				result[f.Alias] = f.Formatter(result)
			}
		} else {
			alias := f.Alias
			if alias == "" {
				alias = f.Source
			}
			result[f.Alias] = flat[prefix+alias]
		}
	}

	return result
}


// ScanJSON выполняет сканирование результатов запроса в формате JSON
func (p Preset) ScanJSON( rows pgx.Rows) ([]any, error) {
	var results []any
	columns := collectColumns(p, "")	

	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range ptrs {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		flat := map[string]any{}
		for i, name := range columns {
			flat[name] = values[i]
		}

		jsonObj := buildNestedJSON(flat, p, "")
		results = append(results, jsonObj)
	}

	return results, nil
}
