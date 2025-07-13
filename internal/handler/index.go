package handler

import (
	"GO_CRM_API/internal/db"
	"GO_CRM_API/internal/preset"
	"context"
	"encoding/json"
	"log"
	"net/http"
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

	query := p.BuildQuery(req.Filters, req.Sorts, req.Offset, req.Limit)
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

	results, err := p.ScanJSON(rows)
	if err != nil {
		http.Error(w, "Scan error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
