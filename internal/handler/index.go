package handler

import (
	"GO_CRM_API/internal/db"
	"GO_CRM_API/internal/preset"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type IndexRequest struct {
	Model   string                 `json:"model"`
	Preset  string                 `json:"preset"`
	Filters map[string]interface{} `json:"filters"`
	Sorts []string								 `json:"sorts"`	
	Offset  uint64                 `json:"offset"`
	Limit   uint64                 `json:"limit"`
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	key := req.Model + "." + req.Preset
	p, err := preset.GetPreset(key)
	if err != nil {
		http.Error(w, "Preset not found: "+key, http.StatusNotFound)
		return
	}

	query := p.BuildQuery(req.Filters, req.Sorts, req.Offset, req.Limit, "")
	sqlStr, args, err := query.ToSql()
	if err != nil {
		http.Error(w, "Failed to build SQL", http.StatusInternalServerError)
		return
	}
	log.Printf("Executing SQL: %s\nARGS: %#v\n", sqlStr, args)
	rows, err := db.Conn.Query(context.Background(), sqlStr, args...)
	if err != nil {
		log.Printf("DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Извлекаем ID-шники родительских записей
	ids, cachedRows, err:= extractPrimaryIDsAndCache(rows, "id")
	if err := rows.Err(); err != nil {
		http.Error(w, "row error", http.StatusInternalServerError)
		return
	}
	
	hasManyData, err := preset.BuildHasManyLoader(context.Background(), p, ids)
	if err != nil {
		http.Error(w, "has_many loader error", http.StatusInternalServerError)
		return
	}
	

	results, err := p.ScanJSON(rows, cachedRows, hasManyData)
	if err != nil {
		http.Error(w, "Scan error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// extractPrimaryIDsAndCache считывает все строки один раз, 
// возвращает ID-шники и кэшированные строки для повторного использования
func extractPrimaryIDsAndCache(rows pgx.Rows, idColumn string) ([]int64, []map[string]any, error) {
	var ids []int64
	var cachedRows []map[string]any

	cols := rows.FieldDescriptions()
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, nil, err
		}

		row := make(map[string]any)
		var idFound bool
		for i, col := range cols {
			name := string(col.Name)
			val := vals[i]
			row[name] = val
			if name == idColumn {
				if id, ok := val.(int64); ok {
					ids = append(ids, id)
					idFound = true
				}
			}
		}

		if !idFound {
			return nil, nil, fmt.Errorf("column %s not found in row", idColumn)
		}

		cachedRows = append(cachedRows, row)
	}

	return ids, cachedRows, rows.Err()
}