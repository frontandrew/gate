package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/frontandrew/gate/internal/pkg/logger"
)

// RecoveryMiddleware восстанавливается после panic и возвращает 500 ошибку
func RecoveryMiddleware(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("Panic recovered", map[string]interface{}{
						"error":      err,
						"stack":      string(debug.Stack()),
						"method":     r.Method,
						"path":       r.URL.Path,
						"remote_addr": r.RemoteAddr,
					})

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":"Internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
