package router

import (
	"GO_CRM_API/internal/handler"
	"net/http"
)

func InitRoutes() {
	http.HandleFunc("/api/index", handler.IndexHandler)
}