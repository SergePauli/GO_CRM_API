package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"GO_CRM_API/internal/db"
	"GO_CRM_API/internal/preset"
)

type CountRequest struct {
	Model   string                 `json:"model"`
	Preset  string                 `json:"preset"`
	Filters map[string]interface{} `json:"filters"`	
}

// CountHandler обрабатывает запросы на подсчет количества записей
// Ожидает JSON с полями Model, Preset, Filters
// Возвращает JSON с полем count, содержащим количество записей
func CountHandler(w http.ResponseWriter, r *http.Request) {
	var req CountRequest
	// Проверяем  запрос
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// Получаем пресет по ключу Model.Preset
	p, err := preset.GetPreset(req.Model + "." + req.Preset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Preset not found: %v", err), http.StatusBadRequest)
		return
	}
	// Строим SQL-запрос для подсчета записей
	query, err := p.BuildCountQuery(req.Filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}

	// Преобразуем запрос в SQL-строку и аргументы
	sqlStr, args, err := query.ToSql()
	if err != nil {
		http.Error(w, fmt.Sprintf("SQL error: %v", err), http.StatusInternalServerError)
		return
	}

	//
	log.Println("SQL for count:", sqlStr)
	log.Println("ARGS:", args)

	// Выполняем запрос к базе данных
	// Используем QueryRow, так как нам нужен только один результат
	// Это оптимально для подсчета количества записей
	row := db.Conn.QueryRow(r.Context(), sqlStr, args...)
	var count int
	if err := row.Scan(&count); err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
		return
	}
	// Возвращаем результат в формате JSON
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}
