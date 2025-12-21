package sgin

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "golang.org/x/text/language"
)

type LoginReq struct {
    Username string `json:"username" label:"用户名" binding:"required,min=3"`
    Password string `json:"password" label:"密码" binding:"required,min=6"`
}

func TestLocalizedValidation(t *testing.T) {
    r := New(Config{
        Locales: []language.Tag{language.Chinese},
    })

    r.POST("/login", func(c *Ctx, req LoginReq) error {
        return nil
    })

    // 辅助函数：检查响应体是否包含中文关键词
    assertContainsChineseKeyword := func(t *testing.T, body string) {
        t.Helper()
        expectedPatterns := []string{"用户名", "不能为空", "必填字段"}
        for _, pattern := range expectedPatterns {
            if strings.Contains(body, pattern) {
                return // 找到任意关键词即通过
            }
        }
        t.Errorf("期望错误信息包含中文关键词，得到 %q", body)
    }

    // 辅助函数：检查响应体是否包含英文关键词（未配置翻译器时）
    assertContainsEnglishKeyword := func(t *testing.T, body string) {
        t.Helper()
        expectedPatterns := []string{"username", "required", "field"}
        for _, pattern := range expectedPatterns {
            if strings.Contains(strings.ToLower(body), strings.ToLower(pattern)) {
                return // 找到任意关键词即通过
            }
        }
        t.Errorf("期望错误信息包含英文关键词，得到 %q", body)
    }

    // 测试中文错误（使用 zh-CN，应该匹配到 zh 翻译器）
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "zh-CN")
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())

    // 测试查询参数 lang=zh-CN
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login?lang=zh-CN", strings.NewReader(`{}`))
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())

    // 测试查询参数 lang=en-US（无效，应回退到默认语言）
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login?lang=en-US", strings.NewReader(`{}`))
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())

    // 测试默认语言（未提供 Accept-Language 头）
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())

    // 测试无效语言头，应回退到默认语言
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "en-US")
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())

    // 测试权重 Accept-Language: zh-CN,zh;q=0.9,en;q=0.8
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())

    // 测试未配置翻译器（零魔法原则）
    r2 := New(Config{})
    r2.POST("/login", func(c *Ctx, req LoginReq) error {
        return nil
    })
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "zh-CN")
    r2.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsEnglishKeyword(t, w.Body.String())
}

func TestSimplifiedLanguageConfig(t *testing.T) {
    // 测试简化配置：使用 Locales 字段
    r := New(Config{
        Locales: []language.Tag{language.Chinese, language.English},
    })

    r.POST("/login", func(c *Ctx, req LoginReq) error {
        return nil
    })

    // 辅助函数：检查响应体是否包含中文关键词
    assertContainsChineseKeyword := func(t *testing.T, body string) {
        t.Helper()
        expectedPatterns := []string{"用户名", "不能为空", "必填字段"}
        for _, pattern := range expectedPatterns {
            if strings.Contains(body, pattern) {
                return // 找到任意关键词即通过
            }
        }
        t.Errorf("期望错误信息包含中文关键词，得到 %q", body)
    }

    // 辅助函数：检查响应体是否包含英文关键词
    assertContainsEnglishKeyword := func(t *testing.T, body string) {
        t.Helper()
        expectedPatterns := []string{"username", "required", "field"}
        for _, pattern := range expectedPatterns {
            if strings.Contains(strings.ToLower(body), strings.ToLower(pattern)) {
                return // 找到任意关键词即通过
            }
        }
        t.Errorf("期望错误信息包含英文关键词，得到 %q", body)
    }

    // 测试中文错误（使用 zh-CN）
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "zh-CN")
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())

    // 测试英文错误（使用 en-US）
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "en-US")
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsEnglishKeyword(t, w.Body.String())

    // 测试语言变体（zh-TW 应回退到 zh）
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "zh-TW")
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())

    // 测试未配置的语言（fr-FR 应回退到默认语言中文）
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "fr-FR")
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    assertContainsChineseKeyword(t, w.Body.String())
}
