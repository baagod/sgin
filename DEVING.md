# Registry 重构开发计划

## 目标
将 `schemaFromType` 重构为独立的 `Registry` 组件，实现职责分离和可测试性提升。

## 核心设计

```go
// registry.go
type Registry struct {
    Namer   func(reflect.Type, string) string `yaml:"-"`
    schemas map[string]*Schema
    exists  map[reflect.Type]bool
    prefix  string
}

func NewRegistry(prefix string, namer func(reflect.Type, string) string) *Registry
func (r *Registry) Register(t reflect.Type, hint ...string) *Schema
func (r *Registry) MarshalYAML() (interface{}, error)
func (r *Registry) FromField(t reflect.Type, isPtr bool, hint ...string) *Schema
```

### Nullable 字段设计

在 `Schema` 结构体中添加 `Nullable` 字段，用于标记可空类型：

```go
// schema.go
type Schema struct {
    Type                 any                `yaml:"type,omitempty"`
    Nullable              bool              `yaml:"-"`  // 不序列化到 YAML
    // ... 其他字段
}

// 用于辅助快速创建可空的基础类型
func NewSchema(typ, format string, nullable bool) *Schema {
	return &{Type: typ, Format: format, Nullable: nullable}
}
```

**序列化逻辑**：当 `Nullable == true` 时，将 `Type` 序列化为 `[type, "null"]` 数组（符合 OpenAPI 3.1 最新标准）。

**注意**：只有基础指针类型（boolean, integer, number, string）才需要 Nullable，对象类型（array, object）不设置。

---

## 阶段一：基础结构搭建（1-2小时）

### 步骤 1.1：创建 registry.go

```go
// 文件：oa/registry.go
package oa

import (
    "reflect"

    "github.com/baagod/sgin/helper"
)

type Registry struct {
    Namer   func(reflect.Type, string) string `yaml:"-"`
    schemas map[string]*Schema
    exists  map[reflect.Type]bool
    prefix  string
}

func NewRegistry(prefix string, namer func(reflect.Type, string) string) *Registry {
    return &Registry{
        Namer:   namer,
        prefix:  prefix,
        schemas: map[string]*Schema{},
        exists:  map[reflect.Type]bool{},
    }
}
```

**验证点**：
- [ ] 结构体字段命名正确
- [ ] `prefix` 参数正确传递
- [ ] `Namer` 签名包含 hint 参数
- [ ] 所有必需的包已导入

### 步骤 1.2：实现 MarshalYAML

```go
// registry.go
func (r *Registry) MarshalYAML() (any, error) {
    return r.schemas, nil
}
```

**验证点**：
- [ ] 实现 `yaml.Marshaler` 接口
- [ ] 输出格式符合 OpenAPI 规范

### 步骤 1.3：修改 Components 结构

```go
// openapi.go
type Components struct {
-   Schemas map[string]*Schema `yaml:"schemas,omitempty"`
+   Schemas *Registry `yaml:"schemas,omitempty"`
}
```

**验证点**：
- [ ] 类型从 `map[string]*Schema` 改为 `*Registry`
- [ ] YAML 标签保持不变

---

## 阶段二：核心 Register 方法（3-4小时）

### 步骤 2.1：实现 Register 方法

```go
// registry.go
// Register 注册类型并返回 Schema（引用或内联）
func (r *Registry) Register(t reflect.Type, hint ...string) *Schema {
    t = helper.DeRef(t)

    // 递归引用检测
    if r.exists[t] {
        if name := r.Namer(t, ""); name != "" {
            return &Schema{Ref: r.prefix + name}
        }
        return nil
    }

    r.exists[t] = true

    isPtr := t.Kind() == reflect.Ptr
    if isPtr {
        t = t.Elem()
    }

    // 基础类型：直接构建 Schema
    switch t.Kind() {
    case reflect.Bool:
        return &Schema{Type: TypeBoolean, Nullable: isPtr}
    case reflect.Int, reflect.Uint:
        if bits.UintSize == 32 {
			return &Schema(Type: TypeInteger, Format: "int32", Nullable: isPtr)
        }
        return &Schema(Type: TypeInteger, Format: "int64", Nullable: isPtr)
    case reflect.Int64, reflect.Uint64:
        return &Schema(Type: TypeInteger, Format: "int64", Nullable: isPtr)
    case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
        return &Schema(Type: TypeInteger, Format: "int32", Nullable: isPtr)
    case reflect.Float32:
        return &Schema(Type: TypeNumber, Format: "float", Nullable: isPtr)
    case reflect.Float64:
        return &Schema(Type: TypeNumber, Format: "double", Nullable: isPtr)
    case reflect.String:
        return &Schema{Type: TypeString, Nullable: isPtr}
    case reflect.Slice, reflect.Array:
        if t.Elem().Kind() == reflect.Uint8 { // []byte 特殊处理
            return &Schema{Type: TypeString, ContentEncoding: "base64"}
        }
        return &Schema{Type: TypeArray, Items: r.Register(t.Elem())}
    case reflect.Map:
        return &Schema{Type: TypeObject, AdditionalProperties: r.Register(t.Elem())}
    case reflect.Struct:
        return r.FromField(t, isPtr, hint...)
    case Interface:
        // 接口可以是任意对象，通过到下面处理。
    default:
        return nil
    }

    // 注册命名结构体到 components
    name := r.Namer(t, "")
    if name != "" {
        // 通过再次调用 Register 获取完整 Schema（不检查 exists）
        schema := r.buildStructSchema(t, isPtr, name, hint...)
        r.schemas[name] = schema
        return &Schema{Ref: r.prefix + name}
    }

    return &Schema{}
}

// buildStructSchema 构建结构体完整 Schema（内部方法，不检查 exists）
func (r *Registry) buildStructSchema(t reflect.Type, isPtr bool, name string, hint ...string) *Schema {
    return r.FromField(t, isPtr, hint...)
}
```

**验证点**：
- [ ] 递归引用正确检测
- [ ] 引用格式使用 `prefix`
- [ ] 避免重复注册同一名称
- [ ] 基础类型 Nullable 正确设置
- [ ] Array/Map 不设置 Nullable
- [ ] Struct 调用 FromField

---

## 阶段三：FromField 方法（4-5小时）

### 步骤 3.1：实现 FromField

```go
// registry.go
// FromField 处理结构体类型，合并字段处理逻辑
func (r *Registry) FromField(t reflect.Type, isPtr bool, hint ...string) *Schema {
    name := r.Namer(t, "")

    // 处理匿名结构体命名提示
    if name == "" && len(hint) > 0 {
        name = hint[0]
    }

    props := make(map[string]*Schema)
    required := []string{}
    fieldSet := make(map[string]struct{})

    // 遍历所有字段（BFS 处理内嵌）
    getFields(t, func(info fieldInfo) {
        f := info.Field

        // 字段遮蔽检查
        if _, ok := fieldSet[f.Name]; ok {
            return
        }
        fieldSet[f.Name] = struct{}{}

        // 解析 JSON 名称
        tag := f.Tag.Get("json")
        if tag == "-" {
            return
        }
        fieldName := f.Name
        if parts := strings.Split(tag, ","); len(parts) > 0 && parts[0] != "" {
            fieldName = parts[0]
        }

        // 生成匿名结构体命名提示
        subHint := ""
        if f.Type.Kind() == reflect.Struct && f.Type.Name() == "" && name != "" {
            subHint = name + f.Name + "Struct"
        }

        // 递归构建 Schema
        schema := r.Register(f.Type, subHint)
        if schema == nil {
            return
        }

        // 引用处理：如果是引用，转换为引用 Schema
        fieldSchema := schema
        if schema.Ref != "" {
            fieldSchema = schema
        }

        // 应用标签
        fieldSchema.Description = f.Tag.Get("doc")
        fieldSchema.ContentEncoding = f.Tag.Get("encoding")

        if v := f.Tag.Get("format"); v != "" {
            // 特殊处理时间格式
            switch v {
            case "2006-01-02":
                fieldSchema.Format = "date"
            case "15:04:05":
                fieldSchema.Format = "time"
            default:
                fieldSchema.Format = v
            }
        }

        if v := f.Tag.Get("default"); v != "" {
            fieldSchema.Default = parseTagValue(v, f.Name, fieldSchema)
        }

        if v := f.Tag.Get("enum"); v != "" {
            ts := fieldSchema
            if ts.Type == TypeArray {
                if ts.Items != nil {
                    ts = ts.Items
                }
            }

            enum := make([]any, 0)
            for _, p := range strings.Split(v, ",") {
                enum = append(enum, parseTagValue(p, f.Name, ts))
            }

            if len(enum) > 0 {
                if fieldSchema.Type == TypeArray && fieldSchema.Items != nil {
                    fieldSchema.Items.Enum = enum
                } else {
                    fieldSchema.Enum = enum
                }
            }
        }

        // 添加到属性
        props[fieldName] = fieldSchema

        // 检查必填
        if strings.Contains(f.Tag.Get("binding"), "required") {
            required = append(required, fieldName)
        }
    })

    s := &Schema{Type: TypeObject, Properties: props}
    if len(required) > 0 {
        s.Required = required
    }

    return s // 不设置 Nullable（Struct 是对象类型）
}
```

**验证点**：
- [ ] 命名逻辑正确
- [ ] 匿名结构体命名提示生效
- [ ] Required 列表正确设置
- [ ] 字段遮蔽逻辑正确
- [ ] JSON 标签 `-` 正确跳过
- [ ] 必填字段正确收集
- [ ] 引用处理正确
- [ ] 所有标签正确应用（包括时间格式）
- [ ] Struct 不设置 Nullable

---

## 阶段四：Config 集成（1小时）

### 步骤 4.1：修改 Config 结构

```go
// config.go
type Config struct {
    *OpenAPI
    tagMap map[string]bool // 暂时不做处理，不处于本次重构范围内。
}
```

**验证点**：
- [ ] 移除 `registry` 字段
- [ ] 移除 `SchemaNamer` 字段
- [ ] 保留 `tagMap`

### 步骤 4.2：修改 New 函数

```go
// config.go
func New(f ...func(*Config)) *Config {
    c := &Config{
        OpenAPI: &OpenAPI{
            OpenAPI: Version,
            Info:    &Info{Title: "APIs", Version: "0.0.1"},
            Paths:   map[string]*PathItem{},
            Components: &Components{
                Schemas: NewRegistry("#/components/schemas/", DefaultSchemaNamer),
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
        },
        tagMap: map[string]bool{},
    }

    if len(f) > 0 {
        f[0](c)
    }

    return c
}
```

**验证点**：
- [ ] Registry 正确初始化（带 prefix）
- [ ] `DefaultSchemaNamer` 正确传递（需调整签名）
- [ ] `tagMap` 初始化

### 步骤 4.3：修改 schemaFromType 委托

```go
// config.go
// schemaFromType 委托给 Registry.Register
func (c *Config) schemaFromType(t reflect.Type, nameHint ...string) *Schema {
    return c.Components.Schemas.Register(t, nameHint...)
}
```

**验证点**：
- [ ] schemaFromType 正确委托
- [ ] 参数传递正确

### 步骤 4.4：调整 DefaultSchemaNamer 签名

```go
// config.go
// DefaultSchemaNamer 根据 "去域名 + 取最后两级" 策略生成名称
func DefaultSchemaNamer(t reflect.Type, hint string) string {
    t = helper.DeRef(t)
    name := t.Name()

    // 如果有 hint（匿名结构体），直接返回
    if name == "" && hint != "" {
        return hint
    }

    // 如果是命名类型，使用原逻辑
    if name != "" {
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
    }

    return name
}
```

**验证点**：
- [ ] `DefaultSchemaNamer` 签名调整为 `func(reflect.Type, string) string`
- [ ] 匿名结构体处理正确
- [ ] 命名类型处理正确

---

## 阶段五：清理和验证（1-2小时）

### 步骤 5.1：删除冗余代码

```go
// config.go
// 删除 schemaFromType 的旧实现（已在 Registry 中实现）
// 保留特殊类型处理逻辑（time、URL、IP 等），后续可集成到 Registry 的 Register 方法中
```

**验证点**：
- [ ] 无用代码全部清理
- [ ] 只保留必要的工具函数

### 步骤 5.2：运行测试

```bash
cd example
go run main.go
curl http://localhost:8080/openapi.yaml
```

**验证点**：
- [ ] OpenAPI 文档生成正确
- [ ] Schema 结构符合预期
- [ ] 无 panic 或错误
- [ ] 输出格式正确

### 步骤 5.3：检查 Nullable 序列化

```bash
# 验证 Nullable 字段是否正确序列化
# 期望：基础类型出现 type: [boolean, null] 或 type: [string, null]
# 期望：对象类型（array, object, struct）只有 type，无 type: [type, null]
grep -A 2 "type:" openapi.yaml
```

**验证点**：
- [ ] 基础类型：`type: [boolean, null]` 或 `type: [string, null]`
- [ ] 对象类型（array, object, struct）：只有 `type: "array"` 或 `type: "object"`
- [ ] 引用类型：只有 `$ref`，无 `type`
- [ ] 无 `nullable: true` 字段（因为标签是 `yaml:"-"`）

---

## 注意事项

1. **Nullable 字段标签**：`yaml:"-"`，不序列化到 YAML，通过自定义序列化器处理
2. **Nullable 序列化逻辑**：当 `Nullable == true` 时，将 `Type` 序列化为 `[type, "null"]` 数组
3. **Nullable 设置范围**：只有基础指针类型（boolean, integer, number, string）才需要 Nullable
4. **Register 简洁性**：Array/Map 处理直接 2-3 行，不拆分单独方法
5. **FromField 内聚性**：合并了 parseFieldMeta、generateFieldHint、isFieldRequired、applyFieldTags 等逻辑
6. **匿名类型处理**：`Register` 返回内联 Schema（非引用），对于匿名结构体
7. **循环引用**：`exists` map 在整个 Schema 构建周期内有效，通过 `buildStructSchema` 二次调用注册完整 Schema
8. **DefaultSchemaNamer 签名**：调整为 `func(reflect.Type, string) string`，hint 参数用于匿名结构体

## 预期收益

| 指标                 | 重构前 | 重构后  | 改进   |
|--------------------|-----|------|------|
| config.go 行数       | 438 | ~180 | -59% |
| schema_fromType 行数 | 195 | ~60  | -69% |
| 可测试函数数             | 0   | 3    | +3   |
| 方法内聚性              | 低   | 高    | 显著   |
| Nullable 支持清晰度     | 低   | 高    | 显著   |
