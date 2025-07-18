package preset

import (
	"context"
	"log"
	"strings"

	"GO_CRM_API/internal/db"

	"github.com/jackc/pgx/v5"
)

// buildHasManyLoader строит карту alias -> related records (по parent_id)
// Используется после основного запроса, когда есть has_many поля в пресете
// buildHasManyLoader выполняет выборку всех has_many полей для переданных parentIDs
func BuildHasManyLoader(ctx context.Context, preset Preset, parentIDs []int64) (map[string]map[int64][]map[string]any, error) {
	result := make(map[string]map[int64][]map[string]any)

	for _, f := range preset.Fields {
		if f.Type != "has_many" || f.NestedPreset == "" {
			continue
		}

		nestedPreset, err := GetPreset(f.NestedPreset)
		if err != nil {
			log.Printf("❌ Invalid nested has_many preset: %s", f.NestedPreset)
			continue
		}

		// строим SQL подзапрос по nestedPreset, фильтруя по parent_id
		builder := nestedPreset.BuildQuery(nil, nil, 0, 0, f.Alias+"_")
		builder = builder.Where(f.FKField + " = ANY(?)", parentIDs)
		query, args, err := builder.ToSql()
		if err != nil {
			return nil, err
		}

		// Выполнение запроса
		rows, err := db.Conn.Query(ctx, query, args...)
		//log.Printf("Executing nestedSQL: %s\nARGS: %#v\n", query, args)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		
		table, err := ScanRowsToMapGroupedBy(rows, strings.ReplaceAll(f.FKField, ".", "_"))
		
		if err != nil {
			return nil, err
		}

		result[f.Alias] = table
	}

	return result, nil
}

// ScanRowsToMapGroupedBy группирует строки по значению поля groupKey (например, parent_id)
func ScanRowsToMapGroupedBy(rows pgx.Rows, groupKey string) (map[int64][]map[string]any, error) {
	grouped := make(map[int64][]map[string]any)

	cols := rows.FieldDescriptions()
	log.Printf("nestedcols: %#v\n", cols)
	for rows.Next() {
		
		values, err := rows.Values()		
		if err != nil {
			return nil, err
		}

		row := make(map[string]any)
		var groupID int64
		for i, col := range cols {
			colName := string(col.Name)
			val := values[i]
			row[colName] = val

			if colName == groupKey {
				if v, ok := val.(int64); ok {
					groupID = v
				}
			}
		}

		if groupID != 0 {
			grouped[groupID] = append(grouped[groupID], row)
		}
	}
	//log.Printf("groupKey: %s\n", groupKey)
	//log.Printf("grouped: %#v\n", grouped)
	return grouped, nil
}
