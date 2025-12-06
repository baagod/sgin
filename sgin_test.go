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
