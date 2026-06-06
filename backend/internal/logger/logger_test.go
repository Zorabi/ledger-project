package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ledger-backend/internal/config"
)

func TestSetupWritesLogFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "app.log")
	output, err := Setup(config.LogConfig{
		Level:      "info",
		File:       true,
		FilePath:   logPath,
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
	})
	if err != nil {
		t.Fatalf("setup logger: %v", err)
	}

	slog.Info("test log entry", "key", "value")
	if err := output.Close(); err != nil {
		t.Fatalf("close logger: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), `"msg":"test log entry"`) {
		t.Fatalf("log entry not found: %s", content)
	}
}
