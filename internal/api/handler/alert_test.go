package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupAlertRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &AlertHandler{}
	r.GET("/alert/channels", h.ListChannels)
	r.POST("/alert/channels", h.CreateChannel)
	r.PUT("/alert/channels/:id", h.UpdateChannel)
	r.DELETE("/alert/channels/:id", h.DeleteChannel)
	r.POST("/alert/channels/:id/test", h.TestChannel)
	r.GET("/alert/rules", h.ListRules)
	r.POST("/alert/rules", h.CreateRule)
	r.PUT("/alert/rules/:id", h.UpdateRule)
	r.DELETE("/alert/rules/:id", h.DeleteRule)
	r.POST("/alert/rules/:id/copy", h.CopyRule)
	r.GET("/alerts", h.ListAlerts)
	r.GET("/alerts/:id", h.GetAlert)
	r.POST("/alerts/:id/acknowledge", h.AcknowledgeAlert)
	r.POST("/alerts/:id/resolve", h.ResolveAlert)
	r.GET("/dashboard/alerts/overview", h.AlertOverview)
	r.GET("/dashboard/alerts/stats", h.AlertStats)
	return r
}

func TestAlertHandler_ListChannels(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("GET", "/alert/channels", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["code"].(float64) != 0 {
		t.Errorf("expected code 0, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	if data["total"].(float64) != 0 {
		t.Errorf("expected total 0, got %v", data["total"])
	}
}

func TestAlertHandler_CreateChannel(t *testing.T) {
	r := setupAlertRouter()

	body := `{"name":"飞书","type":"feishu","endpoint":"https://feishu.example.com","enabled":true}`
	req, _ := http.NewRequest("POST", "/alert/channels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["code"].(float64) != 0 {
		t.Errorf("expected code 0, got %v", resp["code"])
	}
}

func TestAlertHandler_CreateChannel_MissingName(t *testing.T) {
	r := setupAlertRouter()

	body := `{"type":"feishu"}`
	req, _ := http.NewRequest("POST", "/alert/channels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAlertHandler_UpdateChannel(t *testing.T) {
	r := setupAlertRouter()

	body := `{"name":"更新飞书","type":"feishu","enabled":false}`
	req, _ := http.NewRequest("PUT", "/alert/channels/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_DeleteChannel(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("DELETE", "/alert/channels/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["message"] != "删除成功" {
		t.Errorf("expected '删除成功', got %v", resp["message"])
	}
}

func TestAlertHandler_TestChannel(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("POST", "/alert/channels/1/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_ListRules(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("GET", "/alert/rules", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["code"].(float64) != 0 {
		t.Errorf("expected code 0, got %v", resp["code"])
	}
}

func TestAlertHandler_CreateRule(t *testing.T) {
	r := setupAlertRouter()

	body := `{"name":"备份失败告警","level":"P1","enabled":true,"conditions":"{\"event\":\"backup_failed\"}"}`
	req, _ := http.NewRequest("POST", "/alert/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_UpdateRule(t *testing.T) {
	r := setupAlertRouter()

	body := `{"name":"更新规则","level":"P2","enabled":false}`
	req, _ := http.NewRequest("PUT", "/alert/rules/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_DeleteRule(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("DELETE", "/alert/rules/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_CopyRule(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("POST", "/alert/rules/1/copy", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_ListAlerts(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("GET", "/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_GetAlert(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("GET", "/alerts/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})
	if data["level"] != "P2" {
		t.Errorf("expected level P2, got %v", data["level"])
	}
}

func TestAlertHandler_AcknowledgeAlert(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("POST", "/alerts/1/acknowledge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_ResolveAlert(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("POST", "/alerts/1/resolve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAlertHandler_AlertOverview(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("GET", "/dashboard/alerts/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})
	if data["total"] == nil {
		t.Error("expected total field")
	}
	if data["p0"] == nil {
		t.Error("expected p0 field")
	}
}

func TestAlertHandler_AlertStats(t *testing.T) {
	r := setupAlertRouter()
	req, _ := http.NewRequest("GET", "/dashboard/alerts/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})
	if data["today"] == nil {
		t.Error("expected today field")
	}
	if data["week"] == nil {
		t.Error("expected week field")
	}
}
