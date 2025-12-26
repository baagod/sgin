package sgin

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/baagod/sgin/helper"
)

// API 持有 OpenAPI 生成过程中的所有可配置策略
type API struct {
	*OpenAPI
	tagMap map[string]bool
}

func NewAPI(f ...func(*API)) *API {
	c := &API{
		OpenAPI: &OpenAPI{
			OpenAPI: Version,
			Info:    &Info{Title: "APIs", Version: "0.0.1"},
			Paths:   map[string]*PathItem{},
			Components: &Components{
				Schemas: NewRegistry("#/components/schemas/", DefaultSchemaNamer),
				SecuritySchemes: map[string]*SecurityScheme{
					"bearer": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
					"basic":  {Type: "http", Scheme: "basic"},
					"apikey": {Type: "apiKey", Name: "api-key", In: "header"},
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

func (c *API) Register(op *Operation, path, method string, arg *HandleArg) {
	if arg == nil {
		return
	}

	if op.Responses == nil {
		op.Responses = map[string]*ResponseBody{}
	}

	c.parseRequestParams(op, arg.In)
	c.parseResponseBody(op, arg.Out)
	c.registerOperation(op, path, method)
}

// parseRequestParams 解析请求参数 (Path, Query, Header)
func (c *API) parseRequestParams(op *Operation, t reflect.Type) {
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

func (c *API) addParam(op *Operation, name, in, desc string, required bool, t reflect.Type) {
	op.Parameters = append(op.Parameters, &Param{
		Name:        name,
		In:          in,
		Required:    required,
		Description: desc,
		Schema:      c.Components.Schemas.Schema(t),
	})
}

// parseResponseBody 解析响应体
func (c *API) parseResponseBody(op *Operation, t reflect.Type) {
	if t == nil {
		op.Responses["200"] = &ResponseBody{}
		return
	}

	op.Responses["200"] = &ResponseBody{
		Content: map[string]*MediaType{
			"application/json": {
				Schema: c.Components.Schemas.Schema(t),
			},
		},
	}
}

func (c *API) registerOperation(op *Operation, path, method string) {
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
