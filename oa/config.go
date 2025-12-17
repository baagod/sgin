package oa

import (
    "reflect"
    "strings"

    "github.com/baagod/sgin/helper"
)

// DefaultSchemaNamer 根据 “去域名 + 取最后两级” 策略生成名称
func DefaultSchemaNamer(t reflect.Type) string {
    t = helper.DeRef(t)
    name := t.Name()
    parts := strings.Split(t.PkgPath(), "/")

    if len(parts) > 0 && strings.Contains(parts[0], ".") {
        parts = parts[1:]
    }

    if count := len(parts); count >= 2 {
        p1 := toTitle(parts[count-2])
        p2 := toTitle(parts[count-1])
        return p1 + p2 + name
    } else if count == 1 {
        return toTitle(parts[0]) + name
    }

    return name
}

func toTitle(s string) string {
    // 1. 切割字符串
    // strings.FieldsFunc 会遍历字符串 s，每当遇到返回 true 的字符，就把它当做分隔符切断
    words := strings.FieldsFunc(s, func(r rune) bool {
        // 这里定义了分隔符：横线(-)、下划线(_)、点(.)
        return r == '-' || r == '_' || r == '.'
    })

    var sb strings.Builder
    // 2. 遍历切割出来的单词
    for _, w := range words {
        if len(w) > 0 {
            // 3. 首字母大写 + 剩余部分不变
            // strings.ToUpper(w[:1]) 把第一个字母变大写
            // w[1:] 取出后面的字母
            sb.WriteString(strings.ToUpper(w[:1]) + w[1:])
        }
    }
    // 4. 拼接返回
    return sb.String()
}

// Config 持有 OpenAPI 生成过程中的所有可配置策略。
type Config struct {
    // SchemaNamer 是一个函数，用于从 Go 类型生成其在 OpenAPI 组件中的唯一名称。
    SchemaNamer func(t reflect.Type) string
}

func New(c Config, f ...func(*OpenAPI)) *OpenAPI {
    if c.SchemaNamer == nil {
        c.SchemaNamer = DefaultSchemaNamer
    }

    oa := &OpenAPI{
        OpenAPI: Version,
        Info: &Info{
            Title:   "SGin APIs",
            Version: "1.0.0",
        },
        Paths: map[string]*PathItem{},
        Components: &Components{
            Schemas: map[string]*Schema{},
            SecuritySchemes: map[string]*SecurityScheme{
                "bearer": {
                    Type:         "http",
                    Scheme:       "bearer",
                    BearerFormat: "JWT",
                },
                "basic": {
                    Type:   "http",
                    Scheme: "basic",
                },
                "apikey": {
                    Type: "apiKey",
                    Name: "api-key",
                    In:   "header",
                },
            },
        },
        tagMap: map[string]bool{},
        config: c,
    }

    if len(f) > 0 {
        f[0](oa)
    }

    return oa
}
