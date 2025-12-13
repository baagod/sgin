package oa

import (
    "reflect"
    "strings"
)

// defaultSchemaNamer 是一个私有的默认命名函数。
// 它通过组合包路径和类型名称来生成一个全局唯一的 Schema 名称。
func defaultSchemaNamer(t reflect.Type) string {
    fullName := t.PkgPath() + "." + t.Name()
    r := strings.NewReplacer(
        "/", "_",
        ".", "_",
        "-", "_",
    )
    return r.Replace(fullName)
}

// Config 持有 OpenAPI 生成过程中的所有可配置策略。
type Config struct {
    // SchemaNamer 是一个函数，用于从 Go 类型生成其在 OpenAPI 组件中的唯一名称。
    SchemaNamer func(t reflect.Type) string
}

// NewConfig 创建一个带有默认配置的 Config 对象。
func NewConfig() *Config {
    return &Config{
        SchemaNamer: defaultSchemaNamer, // 直接将默认函数赋值
    }
}
