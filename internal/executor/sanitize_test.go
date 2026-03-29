package executor

import (
	"testing"

	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/util"
)

func TestPostgresBuildDumpArgs_SanitizesParams(t *testing.T) {
	task := &model.BackupTask{
		Database: model.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "pass",
			Database: "mydb",
			Params: map[string]string{
				"schema":         "public; rm -rf /",
				"exclude_schema": "$(whoami)",
				"table":          "users|cat /etc/passwd",
				"exclude_table":  "`id`",
			},
		},
		Storage: model.StorageConfig{Path: "/tmp"},
	}
	e := NewPostgresExecutor()
	args := e.buildDumpArgs(task, "/tmp/test.dump")

	for _, arg := range args {
		// No shell metacharacters should appear in any arg
		for _, c := range []string{";", "&", "|", "$", "`"} {
			if arg == "-Fc" || arg == "-f" {
				continue
			}
			if containsStr(arg, c) {
				t.Errorf("arg %q contains dangerous character %q", arg, c)
			}
		}
	}

	// Check specific sanitized values appear
	foundSchema := false
	for i, arg := range args {
		if arg == "-n" && i+1 < len(args) {
			foundSchema = true
			if args[i+1] != "public rm -rf /" {
				t.Errorf("schema param not sanitized: got %q", args[i+1])
			}
		}
	}
	if !foundSchema {
		t.Error("schema param not found in args")
	}
}

func TestMongoDBBuildDumpArgs_SanitizesParams(t *testing.T) {
	task := &model.BackupTask{
		Database: model.DatabaseConfig{
			Host:     "localhost",
			Port:     27017,
			Username: "admin",
			Password: "pass",
			Database: "mydb",
			Params: map[string]string{
				"authenticationDatabase": "admin; DROP TABLE",
				"collection":            "$(cat /etc/passwd)",
				"query":                 "| whoami",
				"readPreference":        "primary&malicious",
			},
		},
		Storage: model.StorageConfig{Path: "/tmp"},
	}
	e := NewMongoDBExecutor()
	args, _ := e.buildDumpArgs(task, "/tmp/out")

	for _, arg := range args {
		for _, c := range []string{";", "&", "|", "$", "`"} {
			if arg == "--gzip" || arg == "--out" {
				continue
			}
			if containsStr(arg, c) {
				t.Errorf("arg %q contains dangerous character %q", arg, c)
			}
		}
	}
}

func TestOracleBuildDumpArgs_SanitizesParams(t *testing.T) {
	// Oracle uses task.Database.Params directly in Backup(), sanitized via util.SanitizeParam
	// Verify the sanitizer works for Oracle-style params
	tests := []struct {
		param string
		want  string
	}{
		{"emp; rm -rf /", "emp rm -rf /"},
		{"$(whoami)", "whoami"},
		{`table:"EMP"`, `table:"EMP"`},
	}
	for _, tt := range tests {
		got := util.SanitizeParam(tt.param)
		if got != tt.want {
			t.Errorf("SanitizeParam(%q) = %q, want %q", tt.param, got, tt.want)
		}
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
