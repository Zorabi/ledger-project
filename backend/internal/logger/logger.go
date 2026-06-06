package logger

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"ledger-backend/internal/config"
)

type Output struct {
	Writer io.Writer
	file   *lumberjack.Logger
}

func Setup(cfg config.LogConfig) (*Output, error) {
	writers := make([]io.Writer, 0, 2)
	if cfg.Console {
		writers = append(writers, os.Stdout)
	}

	var fileWriter *lumberjack.Logger
	if cfg.File {
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0o755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
		fileWriter = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
			LocalTime:  true,
		}
		writers = append(writers, fileWriter)
	}

	if len(writers) == 0 {
		writers = append(writers, io.Discard)
	}

	writer := io.MultiWriter(writers...)
	slog.SetDefault(slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: parseLevel(cfg.Level)})))
	log.SetOutput(writer)

	return &Output{Writer: writer, file: fileWriter}, nil
}

func (o *Output) Close() error {
	if o == nil || o.file == nil {
		return nil
	}
	return o.file.Close()
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
