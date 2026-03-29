package executor

import (
	"os"
	"strings"
	"testing"
)

func TestBuildMinimalEnv_OnlyPathAndLang(t *testing.T) {
	// Set a unique env var in parent process to verify it's not inherited
	t.Setenv("TEST_SECRET_VAR", "should_not_inherit")
	t.Setenv("MYSQL_PWD", "parent_mysql_password")
	t.Setenv("PGPASSWORD", "parent_pg_password")

	env := buildMinimalEnv(nil)

	for _, e := range env {
		if strings.Contains(e, "TEST_SECRET_VAR") {
			t.Error("buildMinimalEnv leaked TEST_SECRET_VAR to child process")
		}
		if strings.Contains(e, "MYSQL_PWD=parent") {
			t.Error("buildMinimalEnv leaked parent MYSQL_PWD to child process")
		}
		if strings.Contains(e, "PGPASSWORD=parent") {
			t.Error("buildMinimalEnv leaked parent PGPASSWORD to child process")
		}
	}

	// Verify PATH is included
	foundPath := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") && strings.Contains(e, "/") {
			foundPath = true
			break
		}
	}
	if !foundPath {
		t.Error("buildMinimalEnv should include PATH")
	}
}

func TestBuildMinimalEnv_WithExtraEnv(t *testing.T) {
	extra := []string{"PGPASSWORD=secret123", "MY_VAR=hello"}
	env := buildMinimalEnv(extra)

	foundPG := false
	foundMyVar := false
	for _, e := range env {
		if e == "PGPASSWORD=secret123" {
			foundPG = true
		}
		if e == "MY_VAR=hello" {
			foundMyVar = true
		}
	}
	if !foundPG {
		t.Error("buildMinimalEnv should include extra PGPASSWORD")
	}
	if !foundMyVar {
		t.Error("buildMinimalEnv should include extra MY_VAR")
	}
}

func TestCreateMySQLDefaultsExtraFile_Security(t *testing.T) {
	password := "test_secret_password_123"
	path, err := createMySQLDefaultsExtraFile(password)
	if err != nil {
		t.Fatalf("createMySQLDefaultsExtraFile failed: %v", err)
	}
	defer os.Remove(path)

	// Verify file permissions are 0600
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("defaults-extra-file permission = %o, want 0600", perm)
	}

	// Verify file contains password
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, password) {
		t.Error("defaults-extra-file should contain the password")
	}
	if !strings.Contains(content, "[client]") {
		t.Error("defaults-extra-file should have [client] section")
	}
}

func TestCreateMySQLDefaultsExtraFile_TempLocation(t *testing.T) {
	path, err := createMySQLDefaultsExtraFile("test")
	if err != nil {
		t.Fatalf("createMySQLDefaultsExtraFile failed: %v", err)
	}
	defer os.Remove(path)

	// Verify it's in /tmp
	if !strings.HasPrefix(path, os.TempDir()) {
		t.Errorf("defaults-extra-file should be in temp dir, got %s", path)
	}
}
