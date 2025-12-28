package sgin

import (
	"fmt"
	"math/bits"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/baagod/sgin/helper"
	"github.com/bytedance/sonic"
)

var (
	mainModule  string
	mainModOnce sync.Once
)

func mainMod() string {
	mainModOnce.Do(func() {
		if info, ok := debug.ReadBuildInfo(); ok {
			mainModule = info.Main.Path
		}
	})
	return mainModule
}

// DefaultSchemaNamer 根据 “去域名 + 取最后两级” 策略生成名称
func DefaultSchemaNamer(t reflect.Type, hint string) string {
	t = helper.Deref(t)

	name := t.Name()
	if name == "" {
		name = hint
	}

	pkgPath := t.PkgPath()
	if pkgPath == "main" {
		pkgPath = ""
	} else {
		pkgPath = strings.TrimPrefix(strings.TrimPrefix(pkgPath, mainMod()), "/")
	}

	parts := strings.Split(pkgPath, "/")
	if len(parts) > 0 && strings.Contains(parts[0], ".") {
		parts = parts[1:]
	}

	if count := len(parts); count >= 2 {
		p1 := helper.CamelCase(parts[count-2])
		p2 := helper.CamelCase(parts[count-1])
		return p1 + p2 + name
	} else if count == 1 {
		return helper.CamelCase(parts[0]) + name
	}

	return name
}

type Registry struct {
	Namer      func(reflect.Type, string) string `yaml:"-"`
	Prefix     string
	schemas    map[string]*Schema
	registered map[reflect.Type]bool
	types      map[string]reflect.Type
}

func NewRegistry(prefix string, namer func(reflect.Type, string) string) *Registry {
	return &Registry{
		Namer:      namer,
		Prefix:     prefix,
		schemas:    map[string]*Schema{},
		registered: map[reflect.Type]bool{},
		types:      map[string]reflect.Type{},
	}
}

func (r *Registry) Schema(t reflect.Type, hint ...string) *Schema {
	nullable := t.Kind() == reflect.Ptr
	if nullable {
		t = t.Elem()
	}

	switch t {
	case timeType:
		return &Schema{Type: TypeString, Format: "date-time", Nullable: nullable}
	case urlType:
		return &Schema{Type: TypeString, Format: "uri", Nullable: nullable}
	case ipType, ipAddrType:
		return &Schema{Type: TypeString, Format: "ipv4", Nullable: nullable}
	case fileHeaderType:
		return &Schema{Type: TypeString, Format: "binary"}
	}

	prefix := t.Name()
	if prefix == "" && len(hint) > 0 {
		prefix = hint[0]
	}

	switch t.Kind() {
	case reflect.Bool:
		return &Schema{Type: TypeBoolean, Nullable: nullable}
	case reflect.Int, reflect.Uint:
		if bits.UintSize == 32 {
			return &Schema{Type: TypeInteger, Format: "int32", Nullable: nullable}
		}
		return &Schema{Type: TypeInteger, Format: "int64", Nullable: nullable}
	case reflect.Int64, reflect.Uint64:
		return &Schema{Type: TypeInteger, Format: "int64", Nullable: nullable}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &Schema{Type: TypeInteger, Format: "int32", Nullable: nullable}
	case reflect.Float32:
		return &Schema{Type: TypeNumber, Format: "float", Nullable: nullable}
	case reflect.Float64:
		return &Schema{Type: TypeNumber, Format: "double", Nullable: nullable}
	case reflect.String:
		return &Schema{Type: TypeString, Nullable: nullable}
	case reflect.Slice, reflect.Array:
		if t == rawMessageType {
			break
		}
		if t.Elem().Kind() == reflect.Uint8 { // []byte 特殊处理
			return &Schema{Type: TypeString, ContentEncoding: "base64"}
		}
		return &Schema{Type: TypeArray, Items: r.Schema(t.Elem(), prefix)}
	case reflect.Map:
		return &Schema{Type: TypeObject, AdditionalProperties: r.Schema(t.Elem(), prefix)}
	case reflect.Struct:
		// 只有 [结构体] 才注册到组件 (Components)，在方法中注册类型并返回。
		return r.Struct(t, hint...)
	case reflect.Interface:
		// 接口可以是任意对象
	default:
		return nil
	}

	return &Schema{} // reflect.Interface 或其他类型
}

func (r *Registry) Field(f reflect.StructField, hint string) (s *Schema) {
	if s = r.Schema(f.Type, hint); s == nil {
		return
	}

	s.Description = f.Tag.Get("doc")
	s.ContentEncoding = f.Tag.Get("encoding")

	if d, ok := f.Tag.Lookup("default"); ok {
		s.Default = helper.Convert(f.Type, f.Name, r.DecodeJSON(d, f.Name, s))
	}

	if format := f.Tag.Get("format"); format != "" {
		switch format {
		case "2006-01-02":
			s.Format = "date"
		case "15:04:05":
			s.Format = "time"
		default:
			s.Format = format
		}
	}

	if values := f.Tag.Get("enum"); values != "" {
		ts := s
		if ts.Type == TypeArray && ts.Items != nil {
			ts = ts.Items
		}

		var enum []any
		for _, p := range strings.Split(values, ",") {
			enum = append(enum, r.DecodeJSON(p, f.Name, ts))
		}

		if len(enum) > 0 {
			if s.Type == TypeArray && s.Items != nil {
				s.Items.Enum = enum
			} else {
				s.Enum = enum
			}
		}
	}

	// 借鉴 Huma 逻辑：如果指针带了 omitempty，且没有显式要求 nullable，
	// 则在文档中将其标记为非 nullable，因为只有 “存在” 和 “缺失” 两种状态。
	if f.Type.Kind() == reflect.Ptr &&
		strings.Contains(f.Tag.Get("json"), "omitempty") &&
		f.Tag.Get("nullable") != "true" {
		s.Nullable = false
	}

	return s
}

func (r *Registry) Struct(t reflect.Type, hint ...string) *Schema {
	if len(hint) == 0 {
		// 如果是匿名结构体，会统一使用 Struct 注册，存在命名冲突，可能需要解决。
		hint = append(hint, "Struct")
	}

	name := r.Namer(t, hint[0])
	if _, ok := r.schemas[name]; ok {
		if _, exist := r.registered[t]; !exist {
			panic(fmt.Errorf("duplicate name: %s, new type: %s, existing type: %s", name, t, r.types[name]))
		}
		return &Schema{Ref: r.Prefix + name}
	}

	// 注册类型以便为递归类型创建 $ref
	s := &Schema{Type: TypeObject}
	r.schemas[name] = s
	r.types[name] = t
	r.registered[t] = true

	var required []string
	props := map[string]*Schema{}
	fieldSet := map[string]struct{}{}

	// 遍历所有字段 (BFS 处理内嵌)
	getFields(t, func(info fieldInfo) {
		f := info.Field

		if _, ok := fieldSet[f.Name]; ok { // 字段遮蔽检查
			return
		}
		fieldSet[f.Name] = struct{}{}

		field := f.Name
		if n := strings.Split(f.Tag.Get("json"), ",")[0]; n != "" {
			field = n // 使用 JSON 字段名称
		}
		if field == "-" {
			return
		}

		fs := r.Field(f, t.Name()+f.Name) // 递归构建 Schema
		if fs == nil {
			return
		}

		if strings.Contains(f.Tag.Get("binding"), "required") {
			required = append(required, field) // 添加必须字段
		}

		props[field] = fs // 添加到属性
	})

	s.Properties = props
	s.Required = required
	return &Schema{Ref: r.Prefix + name}
}

func (r *Registry) MarshalYAML() (any, error) {
	return r.schemas, nil
}

func (r *Registry) Ref(ref string) *Schema {
	if !strings.HasPrefix(ref, r.Prefix) {
		return nil
	}
	return r.schemas[ref[len(r.Prefix):]]
}

// DecodeJSON 根据字段的 Schema 类型，将从 tag 读取的字符串值解析为正确的 Go 类型。
func (r *Registry) DecodeJSON(value, field string, s *Schema) any {
	if s.Ref != "" {
		return r.Ref(s.Ref)
	}

	if s.Type == TypeString {
		return value
	}

	// 特殊情况：字符串数组，带有逗号分隔且无引号。
	if s.Type == TypeArray && s.Items != nil && s.Items.Type == TypeString &&
		len(value) > 0 && value[0] != '[' {
		var values []string
		for _, v := range strings.Split(value, ",") {
			values = append(values, strings.TrimSpace(v))
		}
		return values
	}

	// 对于所有其他类型，尝试使用 JSON 解码器进行解析。
	var result any
	value = strings.TrimSpace(value)
	if err := sonic.ConfigFastest.Unmarshal([]byte(value), &result); err != nil {
		panic(fmt.Errorf("invalid %s tag value '%s' for field '%s': %w", s.Type, value, field, err))
	}

	return result
}
