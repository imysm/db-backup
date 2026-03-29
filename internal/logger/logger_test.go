package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  LogLevel
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"  debug  ", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"", InfoLevel},       // default
		{"unknown", InfoLevel}, // default
		{"warn", WarnLevel},
		{"warning", WarnLevel},
		{"WARN", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestInit(t *testing.T) {
	t.Run("no log file", func(t *testing.T) {
		err := Init("", "info")
		if err != nil {
			t.Errorf("Init() error = %v", err)
		}
		if Info == nil || Warning == nil || Error == nil {
			t.Error("loggers not initialized")
		}
	})

	t.Run("with log file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := tmpDir + "/test.log"
		err := Init(logFile, "debug")
		if err != nil {
			t.Errorf("Init() error = %v", err)
		}
		if Info == nil {
			t.Error("Info logger not initialized")
		}
	})

	t.Run("invalid log file path", func(t *testing.T) {
		err := Init("/nonexistent/dir/test.log", "info")
		if err == nil {
			t.Error("expected error for invalid log file path")
		}
	})
}

func TestLevelWriter(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		writerLevel LogLevel
		shouldWrite bool
	}{
		{"debug written at debug level", "debug", DebugLevel, true},
		{"info written at info level", "info", InfoLevel, true},
		{"debug suppressed at info level", "info", DebugLevel, false},
		{"info suppressed at warn level", "warn", InfoLevel, false},
		{"info suppressed at error level", "error", InfoLevel, false},
		{"warn suppressed at error level", "error", WarnLevel, false},
		{"error written at error level", "error", ErrorLevel, true},
		{"warn suppressed at error level", "error", WarnLevel, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			// Save and restore current loggers
			oldInfo := Info
			oldWarn := Warning
			oldErr := Error

			Init("", tt.level)
			lw := newLevelWriter(&buf, tt.writerLevel)
			_, err := lw.Write([]byte("test log\n"))
			if err != nil {
				t.Errorf("Write() error = %v", err)
			}

			wrote := buf.Len() > 0
			if wrote != tt.shouldWrite {
				t.Errorf("wrote = %v, want %v", wrote, tt.shouldWrite)
			}

			// Restore
			Info = oldInfo
			Warning = oldWarn
			Error = oldErr
		})
	}
}

func TestLoggersOutput(t *testing.T) {
	var buf bytes.Buffer
	Init("", "debug")
	// Replace writers to capture output
	Info.SetOutput(&buf)
	Info.Print("test info message")

	if !strings.Contains(buf.String(), "test info message") {
		t.Errorf("Info logger output missing expected content: %s", buf.String())
	}
}
