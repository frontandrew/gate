package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

// respondJSON отправляет JSON ответ
func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to marshal response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// respondError отправляет JSON ответ с ошибкой
func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{
		"error": message,
	})
}

// getPathParam извлекает параметр из пути URL
// Например: /api/v1/users/123 -> getPathParam(r, "id") = "123"
func getPathParam(r *http.Request, param string) string {
	// Простая реализация для извлечения последнего сегмента пути
	// В реальности лучше использовать роутер с поддержкой параметров (chi, gorilla/mux)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
