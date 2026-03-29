package util

import (
	"strings"
	"testing"
)

func TestValidateBackupPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"", false},
		{"/var/backups/db.sql.gz", true},
		{"../etc/passwd", false},
		{"/safe/path/with spaces.txt", true},
		{"path; DROP TABLE", false},
		{"/data/backup-2024_01.gz", true},
	}

	for _, tt := range tests {
		err := ValidateBackupPath(tt.path)
		if (err == nil) != tt.want {
			t.Errorf("ValidateBackupPath(%q) got err=%v, want valid=%v", tt.path, err, tt.want)
		}
	}
}

func TestValidateDatabaseName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"", false},
		{"mydb", true},
		{"my_db-2024", true},
		{"db; DROP TABLE", false},
		{"my db", false},
		{"db.name", false},
	}

	for _, tt := range tests {
		err := ValidateDatabaseName(tt.name)
		if (err == nil) != tt.want {
			t.Errorf("ValidateDatabaseName(%q) got err=%v, want valid=%v", tt.name, err, tt.want)
		}
	}
}

func TestSanitizeParam(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"normal schema", "public", "public"},
		{"table with underscore", "my_table", "my_table"},
		{"table with dash", "my-table", "my-table"},
		{"table with dot", "schema.table", "schema.table"},
		{"table with star", "schema.*", "schema.*"},
		{"semicolon injection", "public; rm -rf /", "public rm -rf /"},
		{"command substitution", "$(cat /etc/passwd)", "cat /etc/passwd"},
		{"pipe injection", "| cat /etc/passwd", " cat /etc/passwd"},
		{"backtick injection", "`whoami`", "whoami"},
		{"ampersand", "foo&bar", "foobar"},
		{"newline injection", "foo\nbar", "foobar"},
		{"carriage return", "foo\rbar", "foobar"},
		{"parentheses", "foo(bar)", "foobar"},
		{"angle brackets", "foo<bar>baz", "foobarbaz"},
		{"curly braces", "foo{bar}", "foobar"},
		{"dollar sign", "$HOME", "HOME"},
		{"combined attack", "; cat /etc/passwd &", " cat /etc/passwd "},
		{"long string", strings.Repeat("a", 10000), strings.Repeat("a", 10000)},
		{"uri-like", "mongodb://user:pass@host", "mongodb://user:passhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeParam(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeParam(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeWhereClause(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"id > 100", true},
		{"DROP TABLE users", false},
		{"id = 1; DELETE FROM users", false},
		{"1 = 1 -- comment", false},
		{"name = 'test' /* comment */", false},
		{"status = 'active'", true},
	}

	for _, tt := range tests {
		_, err := SanitizeWhereClause(tt.input)
		if (err == nil) != tt.want {
			t.Errorf("SanitizeWhereClause(%q) got err=%v, want valid=%v", tt.input, err, tt.want)
		}
	}
}
