package middleware

import (
	"net/http"
	"time"

	"github.com/frontandrew/gate/internal/pkg/logger"
)

// responseWriter обертка для захвата status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += n
	return n, err
}

// LoggingMiddleware логирует все HTTP запросы
func LoggingMiddleware(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем обертку для response writer
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Обрабатываем запрос
			next.ServeHTTP(rw, r)

			// Логируем
			duration := time.Since(start)
			log.Info("HTTP request", map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status":      rw.statusCode,
				"duration_ms": duration.Milliseconds(),
				"bytes":       rw.written,
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			})
		})
	}
}
