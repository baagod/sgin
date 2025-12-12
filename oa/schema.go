package oa

import (
    "encoding/json"
    "math/bits"
    "net"
    "net/netip"
    "net/url"
    "reflect"
    "time"
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
        s.Type = TypeArray
        s.Items = schemaFromType(t.Elem())
    case reflect.Map:
        s.Type = TypeObject
        s.AdditionalProperties = schemaFromType(t.Elem())
    case reflect.Struct:
        return registerStructSchema(t)
    case reflect.Interface:
        // 接口可以是任意对象
    default:
        return nil // 忽略不支持的类型
    }

    switch s.Type {
    case TypeBoolean, TypeInteger, TypeNumber, TypeString:
        // 作为指针的标量类型默认可为空。
        // 可以通过结构体中的 `nullable:"false"` 字段标签覆盖。
        s.Type = TypeNullable(s.Type)
    }

    return s
}
