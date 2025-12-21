package sgin

import (
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/go-playground/locales/zh"
    "github.com/go-playground/universal-translator"
    "github.com/go-playground/validator/v10"
    tzh "github.com/go-playground/validator/v10/translations/zh"
)

type LoginReq struct {
    Username string `json:"username" label:"用户名" binding:"required,min=3"`
    Password string `json:"password" label:"密码" binding:"required,min=6"`
}

func TestLocalizedValidation(t *testing.T) {
    r := New(Config{
        Locales: []Locale{
            {New: zh.New(), Register: tzh.RegisterDefaultTranslations},
        },
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

func TestLocaleRegistrationFailure(t *testing.T) {
    // 模拟一个返回错误的 Register 函数
    failingRegister := func(*validator.Validate, ut.Translator) error {
        return errors.New("simulated registration failure")
    }

    // 创建一个翻译器，但注册函数会失败
    zhTranslator := zh.New()

    // 创建引擎，包含一个会失败的翻译器和一个成功的翻译器（如果有的话）
    // 但这里我们只配置一个会失败的翻译器，预期结果是没有任何翻译器被注册
    r := New(Config{
        Locales: []Locale{
            {New: zhTranslator, Register: failingRegister},
        },
    })

    r.POST("/login", func(c *Ctx, req LoginReq) error {
        return nil
    })

    // 由于翻译器注册失败，应该没有翻译器可用，错误信息应为英文
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/login", strings.NewReader(`{}`))
    req.Header.Set("Accept-Language", "zh-CN")
    r.engine.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("期望状态码 %d, 得到 %d", http.StatusBadRequest, w.Code)
    }
    // 检查是否为英文错误
    expectedPatterns := []string{"username", "required", "field"}
    found := false
    bodyLower := strings.ToLower(w.Body.String())
    for _, pattern := range expectedPatterns {
        if strings.Contains(bodyLower, pattern) {
            found = true
            break
        }
    }
    if !found {
        t.Errorf("期望错误信息包含英文关键词，得到 %q", w.Body.String())
    }
}
