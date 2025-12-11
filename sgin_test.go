package sgin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 模拟请求结构体
type UserReq struct {
	ID    int    `uri:"id" binding:"required"`
	Type  string `form:"type" default:"guest"`
	Token string `header:"Authorization"`
	Name  string `json:"name"`
}

type UserRes struct {
	ID   int    `json:"id"`
	Info string `json:"info"`
}

// 测试 V2 核心功能
func TestV2Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New()

	// 注册 V2 Handler
	r.POST("/v2/users/:id", func(c *Ctx, req UserReq) (UserRes, error) {
		if req.ID < 0 {
			return UserRes{}, errors.New("invalid id")
		}
		return UserRes{
			ID:   req.ID,
			Info: "Type=" + req.Type + ", Token=" + req.Token + ", Name=" + req.Name,
		}, nil
	})

	// 构造请求
	// Path: /v2/users/100
	// Query: type=admin
	// Header: Authorization=BearerToken
	// Body: {"name": "TestUser"}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v2/users/100?type=admin", strings.NewReader(`{"name": "TestUser"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "BearerToken")

	// 执行
	r.engine.ServeHTTP(w, req)

	// 验证状态码
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d, Body: %s", w.Code, w.Body.String())
	}

	// 验证响应 Body
	var res UserRes
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	if res.ID != 100 {
		t.Errorf("Expected ID 100, got %d", res.ID)
	}
	if !strings.Contains(res.Info, "Type=admin") {
		t.Errorf("Query binding failed: %s", res.Info)
	}
	if !strings.Contains(res.Info, "Token=BearerToken") {
		t.Errorf("Header binding failed: %s", res.Info)
	}
	if !strings.Contains(res.Info, "Name=TestUser") {
		t.Errorf("JSON Body binding failed: %s", res.Info)
	}
}

// 测试 (int, T) 返回值
func TestV2StatusReturn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New()

	r.GET("/status", func(c *Ctx) (int, map[string]string) {
		return 201, map[string]string{"msg": "created"}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil)
	r.engine.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Expected 201, got %d", w.Code)
	}
}

// 测试启动自检 (Fail Fast)
func TestStartupValidation(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid handler signature, but got none")
		}
	}()

	r := New()
	// 注册一个错误的 Handler (3个参数)
	r.GET("/panic", func(c *Ctx, a int, b int) {})
}

// TestOpenAPIGeneration 测试 OpenAPI 文档自动生成
func TestOpenAPIGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 开启 OpenAPI
	r := New(Config{OpenAPI: true})

	// 注册路由
	r.POST("/api/v1/users/:id", func(c *Ctx, req UserReq) (UserRes, error) {
		return UserRes{}, nil
	})

	// 请求 spec
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/openapi.json", nil)
	r.engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200 OK for /openapi.json, got %d", w.Code)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("Failed to unmarshal OpenAPI spec: %v", err)
	}

	// 验证基本信息
	if spec.OpenAPI != "3.2.0" {
		t.Errorf("Expected OpenAPI 3.2.0, got %s", spec.OpenAPI)
	}

	// 验证 Path (注意: Gin的 :id 应该被转换为 {id})
	path := "/api/v1/users/{id}"
	if _, ok := spec.Paths[path]; !ok {
		t.Errorf("Path %s not found in spec. Available: %v", path, spec.Paths)
	}

	op := spec.Paths[path]["post"]
	if op.Responses == nil {
		t.Error("Operation responses not found")
	}

	// 验证 Schema (UserReq)
	// 参数: ID (uri)
	foundID := false
	for _, p := range op.Parameters {
		if p.Name == "id" && p.In == "path" {
			foundID = true
			break
		}
	}
	if !foundID {
		t.Error("Parameter 'id' (in path) not generated")
	}

	// Request Body (Name field from UserReq)
	if op.RequestBody == nil {
		t.Error("RequestBody not generated")
	} else {
		content := op.RequestBody.Content["application/json"]
		if content.Schema == nil || !strings.Contains(content.Schema.Ref, "UserReq") {
			t.Errorf("RequestBody schema ref error, got %v", content.Schema)
		}
	}
}

// TestScalarDocs 测试 Scalar UI 页面
func TestScalarDocs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(Config{OpenAPI: true})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/docs", nil)
	r.engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Errorf("Expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "https://cdn.jsdelivr.net/npm/@scalar/api-reference") {
		t.Error("Scalar CDN link not found in HTML")
	}
	if !strings.Contains(body, "data-url=\"/openapi.json\"") {
		t.Error("data-url configuration not found in HTML")
	}
}

// TestOpenAPIYAML 测试 OpenAPI YAML 文档
func TestOpenAPIYAML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(Config{OpenAPI: true})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/openapi.yaml", nil)
	r.engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200 OK for /openapi.yaml, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/yaml; charset=utf-8" {
		t.Errorf("Expected Content-Type text/yaml, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "openapi: 3.2.0") {
		t.Error("YAML content check failed")
	}
}
