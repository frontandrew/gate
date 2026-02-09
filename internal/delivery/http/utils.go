package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// respondJSON отправляет JSON ответ
func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"Failed to marshal response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
}

// respondError отправляет JSON ответ с ошибкой
func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{
		"error": message,
	})
}

// getPathParam извлекает параметр из пути URL используя chi router context
// Например: /api/v1/users/123 -> getPathParam(r, "id") = "123"
func getPathParam(r *http.Request, param string) string {
	// Сначала пробуем получить из chi router context
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		return chi.URLParam(r, param)
	}

	// Fallback: простая реализация для извлечения последнего сегмента пути
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
