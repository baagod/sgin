package oa

import (
	"math/bits"
	"net/http"
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
	*OpenAPI
	// SchemaNamer 是一个函数，用于从 Go 类型生成其在 OpenAPI 组件中的唯一名称。
	SchemaNamer func(t reflect.Type) string

	tagMap map[string]bool
}

func New(f ...func(*Config)) *Config {
	c := &Config{
		OpenAPI: &OpenAPI{
			OpenAPI: Version,
			Info:    &Info{Title: "APIs", Version: "0.0.1"},
			Paths:   map[string]*PathItem{},
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
		},
		SchemaNamer: DefaultSchemaNamer,
		tagMap:      map[string]bool{},
	}

	if len(f) > 0 {
		f[0](c)
	}

	return c
}

// schemaFromType 递归生成 Schema，支持基础类型、切片、Map 和结构体引用
// nameHint 是可选参数，用于为匿名结构体提供命名提示 (例如: ParentFieldStruct)
func (c *Config) schemaFromType(t reflect.Type, nameHint ...string) *Schema {
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
			s.Items = c.schemaFromType(t.Elem(), nameHint...)
		}
	case reflect.Map:
		s.Type = TypeObject
		s.AdditionalProperties = c.schemaFromType(t.Elem(), nameHint...)
	case reflect.Struct:
		name := t.Name()
		s.Type = TypeObject

		if c.SchemaNamer != nil {
			name = c.SchemaNamer(t)
		}

		// 如果是匿名结构体，尝试使用传入的提示作为名称。
		if name == "" && len(nameHint) > 0 {
			name = nameHint[0]
		}

		// 如果是命名结构体（非匿名，或通过 hint 获得了名字），则处理组件注册和引用逻辑
		if name != "" {
			// 检查该类型是否已在全局组件中注册过
			if _, ok := c.Components.Schemas[name]; ok {
				// 如果已注册，直接返回一个指向该组件的引用，以避免重复定义并处理递归结构
				return &Schema{Ref: "#/components/schemas/" + name}
			}
			// 如果未注册，先创建一个占位符 Schema 并注册，以防止在处理递归字段时陷入无限循环。
			c.Components.Schemas[name] = s
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

			// 2. 为匿名结构体生成命名提示 (ParentName + FieldName + "Struct")
			var subHint string
			// 只有当当前字段是匿名结构体，且父结构体有名称时，才生成提示
			if f.Type.Kind() == reflect.Struct && f.Type.Name() == "" && name != "" {
				subHint = name + f.Name + "Struct"
			}

			// 递归调用 schemaFromType 为当前字段的类型生成 Schema，并传入提示
			fs := c.schemaFromType(f.Type, subHint)
			if fs == nil { // 如果无法为字段类型生成 Schema，则跳过。
				return
			}

			// 3. 解析字段的元数据标签 (doc, format, default, enum)
			fs.Description = f.Tag.Get("doc")
			fs.ContentEncoding = f.Tag.Get("encoding")

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

		// 7. 如果是命名结构体，在填充完所有属性后，最终返回对该组件的引用。
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

// parseRequestParams 解析请求参数 (Path, Query, Header)
func (c *Config) parseRequestParams(op *Operation, t reflect.Type) {
	t = helper.DeRef(t)
	if t.Kind() != reflect.Struct {
		return
	}

	var fields []reflect.StructField

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		desc := f.Tag.Get("doc")
		required := strings.Contains(f.Tag.Get("binding"), "required")

		if tag := f.Tag.Get("uri"); tag != "" {
			c.addParam(op, tag, "path", desc, true, f.Type)
			continue
		}

		if tag := f.Tag.Get("form"); tag != "" {
			c.addParam(op, tag, "query", desc, required, f.Type)
			continue
		}

		if tag := f.Tag.Get("header"); tag != "" {
			c.addParam(op, tag, "header", desc, required, f.Type)
			continue
		}

		if f.Tag.Get("json") != "-" {
			fields = append(fields, f)
		}
	}

	if len(fields) == 0 {
		return
	}

	// 动态创建一个只包含 body 字段的 struct 类型
	// 使用 schemaFromType 一次性生成完整的 BodySchema
	schema := c.schemaFromType(reflect.StructOf(fields))
	if schema == nil {
		return
	}

	op.RequestBody = &RequestBody{
		Content: map[string]*MediaType{
			"application/json": {Schema: schema},
		},
		Required: len(schema.Required) > 0,
	}
}

func (c *Config) addParam(op *Operation, name, in, desc string, required bool, t reflect.Type) {
	op.Parameters = append(op.Parameters, &Param{
		Name:        name,
		In:          in,
		Required:    required,
		Description: desc,
		Schema:      c.schemaFromType(t),
	})
}

// parseResponseBody 解析响应体
func (c *Config) parseResponseBody(op *Operation, t reflect.Type) {
	if t == nil {
		op.Responses["200"] = &Response{}
		return
	}

	op.Responses["200"] = &Response{
		Content: map[string]*MediaType{
			"application/json": {
				Schema: c.schemaFromType(t),
			},
		},
	}
}

func (c *Config) registerOperation(op *Operation, path, method string) {
	method = strings.ToUpper(method)
	apiPath := pathRegex.ReplaceAllString(path, "{$2}")

	if _, ok := c.Paths[apiPath]; !ok {
		c.Paths[apiPath] = &PathItem{}
	}

	switch item := c.Paths[apiPath]; method {
	case http.MethodGet:
		item.Get = op
	case http.MethodHead:
		item.Head = op
	case http.MethodPost:
		item.Post = op
	case http.MethodPut:
		item.Put = op
	case http.MethodPatch:
		item.Patch = op
	case http.MethodDelete:
		item.Delete = op
	case http.MethodOptions:
		item.Options = op
	case http.MethodTrace:
		item.Trace = op
	}

	// 将标签添加到全局列表 (去重)
	for _, tag := range op.Tags {
		if !c.tagMap[tag] {
			c.tagMap[tag] = true
		}
	}
}

func (c *Config) Register(op *Operation, path, method string, handler any) {
	if c == nil {
		return
	}

	if op.Responses == nil {
		op.Responses = map[string]*Response{}
	}

	// 1. 分析入参 (Request)
	t := reflect.TypeOf(handler) // type: sgin.Handler
	if t.NumIn() == 2 {          // func(ctx, input)
		c.parseRequestParams(op, t.In(1))
	}

	// 2. 分析出参 (Response)
	var resType reflect.Type
	for i := 0; i < t.NumOut(); i++ {
		if out := t.Out(i); out.Name() != "error" {
			resType = out
			break
		}
	}

	c.parseResponseBody(op, resType)      // 解析响应体
	c.registerOperation(op, path, method) // 注册操作对象
}
