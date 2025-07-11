package preset

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func collectNestedFields(prefix string, p Preset) []string {
	var fields []string
	for _, f := range p.Fields {
		if f.Type == "preset" || f.Type == "array" {
			nested, err := GetPreset(f.Source)
			if err != nil {
				continue
			}
			subPrefix := prefix + f.Alias + "_"
			fields = append(fields, collectNestedFields(subPrefix, nested)...)
		} else if f.Type != "computed" {
			alias := f.Alias
			if alias == "" {
				alias = f.Source
			}
			fields = append(fields, prefix+alias)
		}
	}
	return fields
}

func (p Preset) ScanJSON( rows pgx.Rows) ([]any, error) {
	var results []any

	columns := []string{}
	for _, f := range p.Fields {
		if f.Type == "preset" {
			parts := strings.Split(f.Source, ".")
			if len(parts) != 2 {
				continue
			}
			key := parts[0]
			nested, err := GetPreset(f.Source)
			if err != nil {
				continue
			}
			for _, nf := range nested.Fields {
				alias := nf.Alias
				if alias == "" {
					alias = nf.Source
				}
				columns = append(columns, fmt.Sprintf("%s_%s", key, alias))
			}
		} else if f.Type != "computed" {
			alias := f.Alias
			if alias == "" {
				alias = f.Source
			}
			columns = append(columns, alias)
		}
	}

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

		// ❗ вот тут магия — рекурсивное построение вложенности
		final := map[string]any{}
		for _, f := range p.Fields {
			if f.Type == "preset" {
				parts := strings.Split(f.Source, ".")
				if len(parts) != 2 {
					continue
				}
				model := parts[0]
				nestedPreset, err := GetPreset(f.Source)
				if err != nil {
					continue
				}

				obj := map[string]any{}
				for _, nf := range nestedPreset.Fields {
					alias := nf.Alias
					if alias == "" {
						alias = nf.Source
					}
					key := model + "_" + alias
					obj[alias] = flat[key]
					delete(flat, key)
				}
				final[model] = obj
			} else if f.Type == "computed" {
				if f.Formatter != nil {
					val := f.Formatter(final)
					final[f.Alias] = val
				}
			} else {
				alias := f.Alias
				if alias == "" {
					alias = f.Source
				}
				final[alias] = flat[alias]
			}
		}
		
		results = append(results, final)
	}

	return results, nil
}
