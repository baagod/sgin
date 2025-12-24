package oa

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/baagod/sgin/helper"
)

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
	schema := c.Components.Schemas.Schema(reflect.StructOf(fields))
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
		Schema:      c.Components.Schemas.Schema(t),
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
				Schema: c.Components.Schemas.Schema(t),
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
