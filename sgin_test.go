package sgin

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/baagod/sgin/oa"
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

// TestOpenAPIGeneration 测试 OpenAPI 文档自动生成
func TestOpenAPIGeneration(t *testing.T) {
    gin.SetMode(gin.TestMode)
    // 开启 OpenAPI
    r := New(Config{OpenAPI: true})

    // 注册路由，使用混合传参配置 OpenAPI
    r.POST("/api/v1/users/:id", func(op *oa.Operation) {
        op.Summary = "创建用户"
        op.Tags = []string{"User"}
    }, func(c *Ctx, req UserReq) (UserRes, error) {
        return UserRes{}, nil
    })

    // 请求 spec
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
    // 简单的字符串包含测试
    if !strings.Contains(body, "openapi: 3.1.1") {
        t.Error("YAML version check failed")
    }
    if !strings.Contains(body, "summary: 创建用户") {
        t.Error("Summary configuration failed")
    }
    if !strings.Contains(body, "- User") { // YAML tags format
        t.Error("Tags configuration failed")
    }
}
