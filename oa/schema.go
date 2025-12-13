package oa

import (
    "encoding/json"
    "fmt"
    "math/bits"
    "net"
    "net/netip"
    "net/url"
    "reflect"
    "strings"
    "time"

    "github.com/baagod/sgin/helper"
)

// JSON Schema 类型常量
const (
    TypeBoolean = "boolean"
    TypeInteger = "integer"
    TypeNumber  = "number"
    TypeString  = "string"
    TypeArray   = "array"
    TypeObject  = "object"
)

func TypeNullable(typ any, t ...reflect.Type) any {
    if len(t) > 0 && t[0].Kind() == reflect.Ptr {
        return []any{typ, "null"}
    }
    return typ
}

// 特殊的 JSON Schema 格式
var (
    timeType       = reflect.TypeOf(time.Time{})
    ipType         = reflect.TypeOf(net.IP{})
    ipAddrType     = reflect.TypeOf(netip.Addr{})
    urlType        = reflect.TypeOf(url.URL{})
    rawMessageType = reflect.TypeOf(json.RawMessage{})
)

type Schema struct {
    Type                 any                 `yaml:"type,omitempty"`
    Title                string              `yaml:"title,omitempty"`
    Description          string              `yaml:"description,omitempty"`
    Ref                  string              `yaml:"$ref,omitempty"`
    Format               string              `yaml:"format,omitempty"`
    ContentEncoding      string              `yaml:"contentEncoding,omitempty"`
    Default              any                 `yaml:"default,omitempty"`
    Examples             []any               `yaml:"examples,omitempty"`
    Items                *Schema             `yaml:"items,omitempty"`                // For arrays
    AdditionalProperties any                 `yaml:"additionalProperties,omitempty"` // Schema or bool
    Properties           map[string]*Schema  `yaml:"properties,omitempty"`
    Enum                 []any               `yaml:"enum,omitempty"`
    MultipleOf           *float64            `yaml:"multipleOf,omitempty"`
    Pattern              string              `yaml:"pattern,omitempty"`
    PatternDescription   string              `yaml:"patternDescription,omitempty"`
    UniqueItems          bool                `yaml:"uniqueItems,omitempty"`
    Required             []string            `yaml:"required,omitempty"`
    ReadOnly             bool                `yaml:"readOnly,omitempty"`
    WriteOnly            bool                `yaml:"writeOnly,omitempty"`
    Deprecated           bool                `yaml:"deprecated,omitempty"`
    Extensions           map[string]any      `yaml:",inline"`
    DependentRequired    map[string][]string `yaml:"dependentRequired,omitempty"`

    OneOf []*Schema `yaml:"oneOf,omitempty"`
    AnyOf []*Schema `yaml:"anyOf,omitempty"`
    AllOf []*Schema `yaml:"allOf,omitempty"`
    Not   *Schema   `yaml:"not,omitempty"`

    // OpenAPI specific fields
    Discriminator *Discriminator `yaml:"discriminator,omitempty"`
}

type Discriminator struct {
    // PropertyName in the payload that will hold the discriminator value.
    // REQUIRED.
    PropertyName string `yaml:"propertyName"`

    // Mapping object to hold mappings between payload values and schema names or
    // references.
    Mapping map[string]string `yaml:"mapping,omitempty"`
}

// fieldInfo 用于存储字段的详细信息，包括其直接父级类型。
// 这在处理复杂的内嵌结构体时非常有用。
type fieldInfo struct {
    Parent reflect.Type
    Field  reflect.StructField
}

// getFields 通过广度优先搜索（BFS）遍历一个类型的所有字段，并在发现每个字段时调用回调函数。
// 它处理内嵌结构体，并通过 visited 集合避免无限递归。
// 使用迭代式 BFS 配合 head 索引，实现高效且清晰的队列操作。
func getFields(t reflect.Type, callback func(info fieldInfo)) {
    // 使用切片模拟队列，并用 head 索引追踪队列头部
    queue := []reflect.Type{t}
    // visited 集合用于防止对同一结构体类型的重复处理
    visited := map[reflect.Type]struct{}{t: {}}

    // 队列处理循环：head 索引在每次迭代中递增，len(queue) 会动态更新
    for head := 0; head < len(queue); head++ {
        currentTyp := queue[head] // 获取当前待处理的类型

        // 遍历当前类型的所有字段
        for i := 0; i < currentTyp.NumField(); i++ {
            f := currentTyp.Field(i)

            // 忽略非导出字段（小写字母开头），因为它们不会被 JSON 序列化
            if !f.IsExported() {
                continue
            }

            // 如果是内嵌字段（匿名字段），则需要进一步处理其内部结构
            if f.Anonymous {
                // 解引用以获取实际类型，因为内嵌字段可能是指针
                embeddedTyp := helper.DeRef(f.Type)

                // 只有当内嵌的是结构体且该类型尚未被访问过时，才将其加入队列等待处理
                if embeddedTyp.Kind() == reflect.Struct {
                    if _, ok := visited[embeddedTyp]; !ok {
                        visited[embeddedTyp] = struct{}{}
                        queue = append(queue, embeddedTyp) // 将新类型加入队列尾部
                    }
                }
                continue // 内嵌字段本身不直接作为 Schema 属性，而是其内部字段会通过回调处理
            }

            // 对于非内嵌的普通字段，执行传入的回调函数
            callback(fieldInfo{Parent: currentTyp, Field: f})
        }
    }
}

// parseTagValue 根据字段的 Schema 类型，将从 tag 读取的字符串值解析为正确的 Go 类型。
// 例如，对于一个 integer 字段，它会将 "123" 解析为数字 123。
func parseTagValue(value, fieldname string, s *Schema) any {
    // 1. 如果基础类型是 string，直接返回原始值，无需解析。
    if s.Type == TypeString {
        return value
    }

    // 特殊情况：字符串数组，带有逗号分隔且无引号。
    if s.Type == TypeArray && s.Items != nil && s.Items.Type == TypeString && value[0] != '[' {
        values := make([]string, 0)
        for _, v := range strings.Split(value, ",") {
            values = append(values, strings.TrimSpace(v))
        }
        return values
    }

    // 2. 对于所有其他类型，尝试使用 JSON 解码器进行解析
    var result any
    value = strings.TrimSpace(value)
    err := json.Unmarshal([]byte(value), &result)
    if err != nil {
        // 返回错误，以便调用者可以决定如何处理（例如，忽略无效的 tag 值）
        panic(fmt.Errorf("invalid %s tag value '%s' for field '%s': %w", s.Type, value, fieldname, err))
    }

    return result
}

// schemaFromType 递归生成 Schema，支持基础类型、切片、Map 和结构体引用
func schemaFromType(t reflect.Type) *Schema {
    isPtr := t.Kind() == reflect.Ptr // 代表可以为 空 (nil) 的数据类型
    if isPtr {
        t = t.Elem()
    }

    // 处理已知标准库类型
    switch t {
    case timeType:
        return &Schema{Type: TypeNullable(TypeString, t), Format: "date-time"}
    case urlType:
        return &Schema{Type: TypeNullable(TypeString, t), Format: "uri"}
    case ipType, ipAddrType:
        return &Schema{Type: TypeNullable(TypeString, t), Format: "ipv4"}
    case rawMessageType:
        return &Schema{}
    }

    s := &Schema{}

    switch t.Kind() {
    case reflect.Bool:
        s.Type = TypeBoolean
    case reflect.Int, reflect.Uint:
        if s.Type = "integer"; bits.UintSize == 32 {
            s.Format = "int32"
        } else {
            s.Format = "int64"
        }
    case reflect.Int64, reflect.Uint64:
        s.Type, s.Format = TypeInteger, "int64"
    case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
        s.Type, s.Format = TypeInteger, "int32"
    case reflect.Float32:
        s.Type, s.Format = TypeNumber, "float"
    case reflect.Float64:
        s.Type, s.Format = TypeNumber, "double"
    case reflect.String:
        s.Type = TypeString
    case reflect.Slice, reflect.Array:
        if t.Elem().Kind() == reflect.Uint8 {
            s.Type = TypeString
            s.ContentEncoding = "base64"
        } else {
            s.Type = TypeArray
            s.Items = schemaFromType(t.Elem())
        }
    case reflect.Map:
        s.Type = TypeObject
        s.AdditionalProperties = schemaFromType(t.Elem())
    case reflect.Struct:
        name := t.Name()
        // 如果是命名结构体（非匿名），则处理组件注册和引用逻辑
        if name != "" {
            // 检查该类型是否已在全局组件中注册过
            if _, ok := Default.Components.Schemas[name]; ok {
                // 如果已注册，直接返回一个指向该组件的引用，以避免重复定义并处理递归结构
                return &Schema{Ref: "#/components/schemas/" + name}
            }
            // 如果未注册，先创建一个占位符 Schema 并注册，以防止在处理递归字段时陷入无限循环。
            // 例如 type Node struct { Next *Node }
            s.Type = TypeObject
            Default.Components.Schemas[name] = s
        } else {
            // 如果是匿名结构体，则直接作为内联对象处理
            s.Type = TypeObject
        }

        // required 用于收集所有必填字段的名称
        var required []string
        // props 用于存储此结构体所有属性的 Schema 定义
        props := make(map[string]*Schema)
        // fieldSet 用于处理 Go 语言的字段遮蔽 (shadowing) 逻辑。
        // 它记录了已经处理过的 Go 字段名 (StructField.Name)，确保外层同名字段优先。
        fieldSet := make(map[string]struct{})

        // 使用 getFields 遍历所有字段，并在回调函数中处理字段的 Schema 生成逻辑
        getFields(t, func(info fieldInfo) {
            f := info.Field // 当前处理的反射字段信息

            // 字段遮蔽判断：如果当前 Go 字段名已被处理过，则跳过
            // (根据 getFields 的 BFS 顺序，这确保了外层同名字段优先)
            if _, ok := fieldSet[f.Name]; ok {
                return
            }
            fieldSet[f.Name] = struct{}{} // 标记当前 Go 字段名已处理

            // 1. 解析字段的 JSON 名称和忽略规则 (json tag)
            fieldName := f.Name // 默认使用 Go 字段名
            tag := f.Tag.Get("json")

            if tag == "-" { // 跳过忽略的字段
                return
            }

            // 解析 json 标签以获取自定义的 JSON 字段名，例如 `json:"my_field,omitempty"`
            if parts := strings.Split(tag, ","); len(parts) > 0 && parts[0] != "" {
                fieldName = parts[0]
            }

            // 2. 递归调用 schemaFromType 为当前字段的类型生成 Schema
            fs := schemaFromType(f.Type)
            if fs == nil { // 如果无法为字段类型生成 Schema，则跳过。
                return
            }

            // 3. 解析字段的元数据标签 (doc, format, default, enum)
            fs.Description = f.Tag.Get("doc")
            if v := f.Tag.Get("format"); v != "" {
                fs.Format = v
            }

            // 使用 parseTagValue 对 default 值进行类型转换
            if v := f.Tag.Get("default"); v != "" {
                fs.Default = parseTagValue(v, f.Name, fs)
            }

            if v := f.Tag.Get("enum"); v != "" {
                ts := fs                  // 目标 schema，对于数组类型，应该验证其子项。
                if ts.Type == TypeArray { // 字段是数组
                    ts = fs.Items
                }

                enum := make([]any, 0, t.NumField())
                for _, p := range strings.Split(v, ",") { // 对每个枚举值进行解析
                    enum = append(enum, parseTagValue(p, f.Name, ts))
                }

                if len(enum) > 0 {
                    if fs.Type == TypeArray { // 如果字段是数组，枚举应用于子项。
                        if fs.Items != nil {
                            fs.Items.Enum = enum
                        }
                    } else {
                        fs.Enum = enum
                    }
                }
            }

            // 4. 将生成的字段 Schema 添加到属性 map 中
            props[fieldName] = fs

            // 5. 处理字段的必填逻辑
            if strings.Contains(f.Tag.Get("binding"), "required") {
                required = append(required, fieldName)
            }
        })

        // 6. 将收集到的属性和必填列表赋值给主 Schema 对象
        if len(props) > 0 {
            s.Properties = props
        }

        if len(required) > 0 {
            s.Required = required
        }

        // 7. 如果是命名结构体，在填充完所有属性后，最终返回对该组件的引用
        if name != "" {
            return &Schema{Ref: "#/components/schemas/" + name}
        }
    case reflect.Interface:
        // 接口可以是任意对象
    default:
        return nil // 忽略不支持的类型
    }

    switch s.Type {
    case TypeBoolean, TypeInteger, TypeNumber, TypeString:
        // 作为指针的标量类型默认可为空。
        // 可以通过结构体中的 `nullable:"false"` 字段标签覆盖。
        if isPtr {
            s.Type = []any{s.Type, "null"}
        }
    }

    return s
}
