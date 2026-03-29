package audit

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRemoveAPIv1Prefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/v1/jobs/123", "jobs/123"},
		{"/api/v1/users", "users"},
		{"/api/v1/", "/api/v1/"}, // len == prefix len, not >, so not removed
		{"/api/v1", "/api/v1"},   // len == prefix len, not >, so not removed
		{"/jobs/123", "/jobs/123"},
		{"/api/v2/jobs", "/api/v2/jobs"},
		{"", ""},
		{"/api/v10/jobs", "/api/v10/jobs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := removeAPIv1Prefix(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimSlashes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/jobs/", "jobs"},
		{"/jobs", "jobs"},
		{"///jobs///", "jobs"},
		{"jobs", "jobs"},
		{"", ""},
		{"/", ""},
		{"///", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := trimSlashes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"jobs/123", []string{"jobs", "123"}},
		{"users", []string{"users"}},
		{"api/v1/jobs", []string{"api", "v1", "jobs"}},
		{"", nil},
		{"/", nil},
		{"///", []string{}}, // trimSlashes removes all -> "" -> empty slice
		{"/jobs/123/", []string{"jobs", "123"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeResourceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"jobs", "job"},
		{"records", "record"},
		{"templates", "template"},
		{"verify", "verify"},
		{"restore", "restore"},
		{"merge", "merge"},
		{"stats", "stats"},
		{"users", "user"},
		{"roles", "role"},
		{"tenants", "tenant"},
		{"settings", "setting"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeResourceName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseResource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		path           string
		params         gin.Params
		wantResource   string
		wantResourceID *uint
	}{
		{
			name:           "jobs with id",
			path:           "/api/v1/jobs/123",
			params:         gin.Params{{Key: "id", Value: "123"}},
			wantResource:   "job",
			wantResourceID: uintPtr(123),
		},
		{
			name:           "users without id",
			path:           "/api/v1/users",
			params:         gin.Params{},
			wantResource:   "user",
			wantResourceID: nil,
		},
		{
			name:           "templates with id",
			path:           "/api/v1/templates/5",
			params:         gin.Params{{Key: "id", Value: "5"}},
			wantResource:   "template",
			wantResourceID: uintPtr(5),
		},
		{
			name:           "restore",
			path:           "/api/v1/restore/7",
			params:         gin.Params{{Key: "id", Value: "7"}},
			wantResource:   "restore",
			wantResourceID: uintPtr(7),
		},
		{
			name:           "tenants",
			path:           "/api/v1/tenants/1",
			params:         gin.Params{{Key: "id", Value: "1"}},
			wantResource:   "tenant",
			wantResourceID: uintPtr(1),
		},
		{
			name:           "empty path",
			path:           "",
			params:         gin.Params{},
			wantResource:   "unknown",
			wantResourceID: nil,
		},
		{
			name:           "no id in path",
			path:           "/api/v1/stats",
			params:         gin.Params{},
			wantResource:   "stats",
			wantResourceID: nil,
		},
		{
			name:           "roles with id",
			path:           "/api/v1/roles/9",
			params:         gin.Params{{Key: "id", Value: "9"}},
			wantResource:   "role",
			wantResourceID: uintPtr(9),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, resourceID := parseResource(tt.path, tt.params)
			assert.Equal(t, tt.wantResource, resource)
			if tt.wantResourceID == nil {
				assert.Nil(t, resourceID)
			} else {
				assert.NotNil(t, resourceID)
				assert.Equal(t, *tt.wantResourceID, *resourceID)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	details := NewAuditDetails()
	details.Before["name"] = "old"
	details.After["name"] = "new"
	details.Method = "POST"
	details.Path = "/api/v1/jobs"

	json := details.ToJSON()
	assert.Contains(t, json, `"before"`)
	assert.Contains(t, json, `"after"`)
	assert.Contains(t, json, `"method":"POST"`)
	assert.Contains(t, json, `"path":"/api/v1/jobs"`)

	// Empty details
	empty := &AuditDetails{}
	assert.Equal(t, "{}", empty.ToJSON())
}
