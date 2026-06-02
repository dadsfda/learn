package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 测试 GET /ping
func TestPing(t *testing.T) {
	router := SetupRouter()

	req, _ := http.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200，实际 %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["message"] != "pong" {
		t.Errorf("期望 message=pong，实际 %s", resp["message"])
	}
}

// 测试 GET /health
func TestHealth(t *testing.T) {
	router := SetupRouter()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}
}

// 测试 POST /login - JSON 格式
func TestLoginJSON(t *testing.T) {
	router := SetupRouter()

	// 正确的用户名密码
	body := strings.NewReader(`{"username":"admin","password":"123456"}`)
	req, _ := http.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["token"] == "" {
		t.Error("期望返回 token")
	}
}

// 测试 POST /login - 错误密码
func TestLoginWrongPassword(t *testing.T) {
	router := SetupRouter()

	body := strings.NewReader(`{"username":"admin","password":"wrong"}`)
	req, _ := http.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("错误密码应返回 401，实际 %d", w.Code)
	}
}

// 测试 POST /login - 缺少字段
func TestLoginMissingFields(t *testing.T) {
	router := SetupRouter()

	body := strings.NewReader(`{"username":"admin"}`)
	req, _ := http.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("缺少字段应返回 400，实际 %d", w.Code)
	}
}

// 测试 POST /login - 表单格式
func TestLoginForm(t *testing.T) {
	router := SetupRouter()

	form := url.Values{}
	form.Set("username", "admin")
	form.Set("password", "123456")

	req, _ := http.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("表单登录期望 200，实际 %d", w.Code)
	}
}

// 测试路径参数
func TestGetUser(t *testing.T) {
	router := SetupRouter()

	req, _ := http.NewRequest("GET", "/users/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] != "42" {
		t.Errorf("期望 id=42，实际 %v", resp["id"])
	}
}
