# Registry 重构开发计划

## 目标
将 `schemaFromType` 重构为独立的 `Registry` 组件，实现职责分离和可测试性提升。

## 核心设计

```go
// schema_registry.go
type Registry struct {
    schemas map[string]*Schema
    namer   func(reflect.Type, string) string
    exists  map[reflect.Type]bool
    prefix  string
}

func NewRegistry(namer func(reflect.Type) string) *Registry
func (r *Registry) Register(t reflect.Type, hint ...string) string
func (r *Registry) MarshalYAML() (any, error)
func (r *Registry) buildSchema(t reflect.Type, hint ...string) *Schema
```

---

## 阶段一：基础结构搭建（2-3小时）

### 步骤 1.1：创建 schema_registry.go
```go
// 文件：oa/schema_registry.go
package oa

import (
    "reflect"

    "github.com/baagod/sgin/helper"
)

type Registry struct {
    schemas map[string]*Schema
    namer   func(reflect.Type) string
    exists  map[reflect.Type]bool
    prefix  string
}

func NewRegistry(namer func(reflect.Type) string) *Registry {
    return &Registry{
        schemas: make(map[string]*Schema),
        namer:   namer,
        exists:  make(map[reflect.Type]bool),
        prefix:  "#/components/schemas/",
    }
}
```

**验证点**：
- [ ] 结构体字段命名正确
- [ ] `prefix` 默认值设置正确
- [ ] 所有必需的包已导入

### 步骤 1.2：实现 MarshalYAML
```go
// schema_registry.go
func (r *Registry) MarshalYAML() (interface{}, error) {
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

## 阶段二：核心逻辑迁移（4-5小时）

### 步骤 2.1：实现 Register 方法框架
```go
// schema_registry.go
func (r *Registry) Register(t reflect.Type, hint ...string) string {
    t = helper.DeRef(t)

    // 递归引用检测
    if r.exists[t] {
        if name := r.namer(t); name != "" {
            return r.prefix + name
        }
        return ""
    }

    r.exists[t] = true

    // 构建 Schema
    schema := r.buildSchema(t, hint...)

    // 注册命名结构体
    if name := r.namer(t); name != "" {
        if r.schemas[name] == nil {
            r.schemas[name] = schema
        }
        return r.prefix + name
    }

    return ""
}
```

**验证点**：
- [ ] 递归引用正确检测
- [ ] 引用格式使用 `prefix`
- [ ] 避免重复注册同一名称

### 步骤 2.2：迁移基础类型处理
```go
// schema_registry.go
func (r *Registry) buildSchema(t reflect.Type, hint ...string) *Schema {
    isPtr := t.Kind() == reflect.Ptr
    if isPtr {
        t = t.Elem()
    }

    // 暂时移除特殊类型处理（time、URL、IP）

    switch t.Kind() {
    case reflect.Bool:
        return &Schema{Type: TypeBoolean}
    case reflect.Int, reflect.Uint:
        s := &Schema{Type: TypeInteger}
        if bits.UintSize == 32 {
            s.Format = "int32"
        } else {
            s.Format = "int64"
        }
        return r.applyNullable(s, isPtr)
    case reflect.Int64, reflect.Uint64:
        return r.applyNullable(&Schema{Type: TypeInteger, Format: "int64"}, isPtr)
    case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
        return r.applyNullable(&Schema{Type: TypeInteger, Format: "int32"}, isPtr)
    case reflect.Float32:
        return r.applyNullable(&Schema{Type: TypeNumber, Format: "float"}, isPtr)
    case reflect.Float64:
        return r.applyNullable(&Schema{Type: TypeNumber, Format: "double"}, isPtr)
    case reflect.String:
        return r.applyNullable(&Schema{Type: TypeString}, isPtr)
    case reflect.Slice, reflect.Array:
        return r.buildArray(t, isPtr)
    case reflect.Map:
        return r.buildMap(t, isPtr)
    case reflect.Struct:
        return r.buildStruct(t, isPtr, hint...)
    default:
        return nil
    }
}

func (r *Registry) applyNullable(s *Schema, isPtr bool) *Schema {
    if isPtr {
        s.Type = []any{s.Type, "null"}
    }
    return s
}
```

**验证点**：
- [ ] 所有基础类型正确映射
- [ ] Nullable 逻辑正确应用
- [ ] 特殊类型暂时跳过

### 步骤 2.3：实现 buildArray
```go
// schema_registry.go
func (r *Registry) buildArray(t reflect.Type, isPtr bool) *Schema {
    if t.Elem().Kind() == reflect.Uint8 {
        s := &Schema{Type: TypeString, ContentEncoding: "base64"}
        return r.applyNullable(s, isPtr)
    }

    s := &Schema{Type: TypeArray}
    s.Items = r.Register(t.Elem())
    return r.applyNullable(s, isPtr)
}
```

**验证点**：
- [ ] `[]byte` 正确识别为 base64 字符串
- [ ] 数组 Items 递归调用 `Register`

### 步骤 2.4：实现 buildMap
```go
// schema_registry.go
func (r *Registry) buildMap(t reflect.Type, isPtr bool) *Schema {
    s := &Schema{Type: TypeObject}
    s.AdditionalProperties = r.Register(t.Elem())
    return r.applyNullable(s, isPtr)
}
```

**验证点**：
- [ ] Map 类型正确映射为 Object
- [ ] AdditionalProperties 递归调用 `Register`

---

## 阶段三：Struct 处理逻辑迁移（6-8小时）

### 步骤 3.1：实现 buildStruct 框架
```go
// schema_registry.go
func (r *Registry) buildStruct(t reflect.Type, isPtr bool, hint ...string) *Schema {
    name := r.namer(t)

    // 处理匿名结构体命名提示
    if name == "" && len(hint) > 0 {
        name = hint[0]
    }

    props, required := r.collectStructFields(t, name)

    s := &Schema{Type: TypeObject, Properties: props}
    if len(required) > 0 {
        s.Required = required
    }

    return r.applyNullable(s, isPtr)
}
```

**验证点**：
- [ ] 命名逻辑正确
- [ ] 匿名结构体命名提示生效
- [ ] Required 列表正确设置

### 步骤 3.2：实现 collectStructFields
```go
// schema_registry.go
func (r *Registry) collectStructFields(t reflect.Type, parentName string) (map[string]*Schema, []string) {
    props := make(map[string]*Schema)
    required := []string{}
    fieldSet := make(map[string]struct{})

    getFields(t, func(info fieldInfo) {
        f := info.Field

        // 字段遮蔽检查
        if _, ok := fieldSet[f.Name]; ok {
            return
        }
        fieldSet[f.Name] = struct{}{}

        // 解析字段元信息
        fieldName, skip := r.parseFieldMeta(f)
        if skip {
            return
        }

        // 生成命名提示
        subHint := r.generateFieldHint(f, parentName)

        // 递归构建 Schema
        schema := r.Register(f.Type, subHint)
        if schema == "" {
            return
        }
        fieldSchema := r.getFieldSchema(schema)

        // 应用标签
        r.applyFieldTags(fieldSchema, f)

        // 添加到属性
        props[fieldName] = fieldSchema

        // 检查必填
        if r.isFieldRequired(f) {
            required = append(required, fieldName)
        }
    })

    return props, required
}
```

**验证点**：
- [ ] 字段遮蔽逻辑正确
- [ ] JSON 标签 `-` 正确跳过
- [ ] 必填字段正确收集

### 步骤 3.3：辅助方法实现
```go
// schema_registry.go

// parseFieldMeta 解析 JSON 名称
func (r *Registry) parseFieldMeta(f reflect.StructField) (string, bool) {
    tag := f.Tag.Get("json")
    if tag == "-" {
        return "", true
    }

    fieldName := f.Name
    if parts := strings.Split(tag, ","); len(parts) > 0 && parts[0] != "" {
        fieldName = parts[0]
    }
    return fieldName, false
}

// generateFieldHint 生成匿名结构体命名提示
func (r *Registry) generateFieldHint(f reflect.StructField, parentName string) string {
    if f.Type.Kind() == reflect.Struct && f.Type.Name() == "" && parentName != "" {
        return parentName + f.Name + "Struct"
    }
    return ""
}

// isFieldRequired 检查字段是否必填
func (r *Registry) isFieldRequired(f reflect.StructField) bool {
    return strings.Contains(f.Tag.Get("binding"), "required")
}

// getFieldSchema 从引用字符串获取实际 Schema
func (r *Registry) getFieldSchema(ref string) *Schema {
    if strings.HasPrefix(ref, r.prefix) {
        return &Schema{Ref: ref}
    }
    return nil
}
```

**验证点**：
- [ ] JSON 标签解析正确
- [ ] 匿名结构体提示生成逻辑正确

### 步骤 3.4：实现 applyFieldTags
```go
// schema_registry.go
func (r *Registry) applyFieldTags(schema *Schema, f reflect.StructField) {
    schema.Description = f.Tag.Get("doc")
    schema.ContentEncoding = f.Tag.Get("encoding")

    if v := f.Tag.Get("format"); v != "" {
        schema.Format = v
    }

    if v := f.Tag.Get("default"); v != "" {
        schema.Default = parseTagValue(v, f.Name, schema)
    }

    if v := f.Tag.Get("enum"); v != "" {
        r.applyEnumTag(schema, f, v)
    }
}

// applyEnumTag 处理枚举标签
func (r *Registry) applyEnumTag(schema *Schema, f reflect.StructField, enumStr string) {
    ts := schema
    if ts.Type == TypeArray {
        if ts.Items != nil {
            ts = ts.Items
        }
    }

    enum := make([]any, 0)
    for _, p := range strings.Split(enumStr, ",") {
        enum = append(enum, parseTagValue(p, f.Name, ts))
    }

    if len(enum) > 0 {
        if schema.Type == TypeArray && schema.Items != nil {
            schema.Items.Enum = enum
        } else {
            schema.Enum = enum
        }
    }
}
```

**验证点**：
- [ ] 所有标签正确应用
- [ ] 枚举处理支持数组和非数组

---

## 阶段四：Config 集成（1-2小时）

### 步骤 4.1：修改 Config 结构
```go
// config.go
type Config struct {
    *OpenAPI
    SchemaNamer func(reflect.Type) string
-   tagMap     map[string]bool
+   registry    *Registry
}
```

**验证点**：
- [ ] 移除 `tagMap`
- [ ] 添加 `registry` 字段

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
                Schemas: NewRegistry(DefaultSchemaNamer),
                SecuritySchemes: map[string]*SecurityScheme{...},
            },
        },
        SchemaNamer: DefaultSchemaNamer,
    }

    c.registry = c.Components.Schemas

    if len(f) > 0 {
        f[0](c)
    }

    return c
}
```

**验证点**：
- [ ] Registry 正确初始化
- [ ] Config.registry 指向 Components.Schemas

### 步骤 4.3：修改 schemaFromType 调用
```go
// config.go
func (c *Config) schemaFromType(t reflect.Type, hint ...string) *Schema {
    ref := c.registry.Register(t, hint...)

    if strings.HasPrefix(ref, c.registry.prefix) {
        return &Schema{Ref: ref}
    }

    return nil
}
```

**验证点**：
- [ ] schemaFromType 调用正确委托
- [ ] 引用处理正确

### 步骤 4.4：移除 tagMap 相关代码
```go
// config.go - registerOperation 方法
func (c *Config) registerOperation(op *Operation, path, method string) {
    // ... 其他代码 ...

-   for _, tag := range op.Tags {
-       if !c.tagMap[tag] {
-           c.tagMap[tag] = true
-       }
-   }
}
```

**验证点**：
- [ ] tagMap 引用全部移除
- [ ] 标签去重逻辑评估必要性

---

## 阶段五：清理和验证（2-3小时）

### 步骤 5.1：删除冗余代码
```go
// config.go
- // 删除 schemaFromType 的旧实现（已在 Registry 中）
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

### 步骤 5.3：对比验证
```bash
git stash
go run example/main.go > before.yaml
git stash pop
go run example/main.go > after.yaml
diff before.yaml after.yaml
```

**验证点**：
- [ ] 输出完全一致（或等价）
- [ ] 无功能退化

---

## 注意事项

1. **匿名类型处理**：当前 `Register` 返回空字符串表示匿名类型，需要 `schemaFromType` 返回内联 Schema
2. **特殊类型**：暂时移除 `time`、`URL`、`IP` 等处理，后续可添加 `specialTypeHandler`
3. **循环引用**：`exists` map 在整个 Schema 构建周期内有效，无需 `Reset()`

## 预期收益

| 指标 | 重构前 | 重构后 | 改进 |
|------|--------|--------|------|
| config.go 行数 | 438 | ~200 | -54% |
| schemaFromType 行数 | 195 | ~50 | -74% |
| 可测试函数数 | 0 | 15 | +15 |
| 职责分离 | 低 | 高 | 显著 |
