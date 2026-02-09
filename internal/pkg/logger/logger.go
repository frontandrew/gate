package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger - интерфейс для логирования
type Logger interface {
	Debug(msg string, fields ...map[string]interface{})
	Info(msg string, fields ...map[string]interface{})
	Warn(msg string, fields ...map[string]interface{})
	Error(msg string, fields ...map[string]interface{})
	Fatal(msg string, fields ...map[string]interface{})
	With(key string, value interface{}) Logger
}

// zerologLogger - реализация Logger на основе zerolog
type zerologLogger struct {
	logger zerolog.Logger
}

// New создает новый logger с заданным уровнем и форматом
func New(level, format, output string) Logger {
	// Парсим уровень логирования
	logLevel := parseLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	// Настраиваем вывод
	var writer io.Writer
	if output == "stdout" || output == "" {
		writer = os.Stdout
	} else {
		// Можно добавить запись в файл
		writer = os.Stdout
	}

	// Настраиваем формат
	if format == "console" {
		writer = zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
		}
	}

	logger := zerolog.New(writer).
		With().
		Timestamp().
		Caller().
		Logger()

	return &zerologLogger{logger: logger}
}

func (l *zerologLogger) Debug(msg string, fields ...map[string]interface{}) {
	event := l.logger.Debug()
	l.addFields(event, fields)
	event.Msg(msg)
}

func (l *zerologLogger) Info(msg string, fields ...map[string]interface{}) {
	event := l.logger.Info()
	l.addFields(event, fields)
	event.Msg(msg)
}

func (l *zerologLogger) Warn(msg string, fields ...map[string]interface{}) {
	event := l.logger.Warn()
	l.addFields(event, fields)
	event.Msg(msg)
}

func (l *zerologLogger) Error(msg string, fields ...map[string]interface{}) {
	event := l.logger.Error()
	l.addFields(event, fields)
	event.Msg(msg)
}

func (l *zerologLogger) Fatal(msg string, fields ...map[string]interface{}) {
	event := l.logger.Fatal()
	l.addFields(event, fields)
	event.Msg(msg)
}

func (l *zerologLogger) With(key string, value interface{}) Logger {
	newLogger := l.logger.With().Interface(key, value).Logger()
	return &zerologLogger{logger: newLogger}
}

// addFields добавляет дополнительные поля к событию логирования
func (l *zerologLogger) addFields(event *zerolog.Event, fields []map[string]interface{}) {
	if len(fields) > 0 {
		for _, fieldMap := range fields {
			for key, value := range fieldMap {
				event.Interface(key, value)
			}
		}
	}
}

// parseLevel преобразует строковое значение уровня в zerolog.Level
func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// SetGlobalLogger устанавливает глобальный logger
func SetGlobalLogger(logger Logger) {
	if zl, ok := logger.(*zerologLogger); ok {
		log.Logger = zl.logger
	}
}

// NewDevelopment creates a logger suitable for development/testing
func NewDevelopment() Logger {
	return New("debug", "console", "stdout")
}

// NewNoop creates a noop logger that discards all log messages
func NewNoop() Logger {
	logger := zerolog.New(io.Discard)
	return &zerologLogger{logger: logger}
}
