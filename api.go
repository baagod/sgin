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
			tagMap: map[string]bool{},
		},
	}

	if len(f) > 0 {
		f[0](c)
	}

	return c
}

// Register 将处理器的元数据 (输入/输出类型) 注册到指定的 Operation 中
func (c *API) Register(op *Operation, path, method string, arg *HandleArg) {
	if arg == nil {
		return
	}

	// 确保响应对象已初始化，避免后续注入时出现 nil map
	if op.Responses == nil {
		op.Responses = map[string]*ResponseBody{}
	}

	c.parseRequestParams(op, arg.In)      // 解析结构体标签并映射为请求参数或 RequestBody
	c.parseResponseBody(op, arg.Out)      // 解析返回类型并映射为 ResponseBody
	c.registerOperation(op, path, method) // 将配置好的 Operation 绑定到 OpenAPI 路径树中
}

// parseRequestParams 解析输入结构体的标签 (uri, form, header, json)，并映射为 OpenAPI 的参数或请求体。
func (c *API) parseRequestParams(op *Operation, t reflect.Type) {
	t = helper.DeRef(t)
	if t.Kind() != reflect.Struct {
		return
	}

	var jsonFields []reflect.StructField // 用于收集映射到 JSON Body 的字段

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		desc := f.Tag.Get("doc")                                       // 从 doc 标签获取字段描述
		required := strings.Contains(f.Tag.Get("binding"), "required") // 检查是否标记为必填

		// 1. 处理路径参数 (uri 标签) -> 映射至 OpenAPI path 参数
		if tag := f.Tag.Get("uri"); tag != "" {
			// 根据 OpenAPI 规范，路径参数必须是必填的 (required: true)
			c.addParam(op, tag, "path", desc, true, f.Type)
			continue
		}

		// 2. 处理查询参数 (form 标签) -> 映射至 OpenAPI query 参数
		// Gin 的 form 标签同时支持 URL 查询字符串和表单 Body，此处统一映射为 query
		if tag := f.Tag.Get("form"); tag != "" {
			c.addParam(op, tag, "query", desc, required, f.Type)
			continue
		}

		// 3. 处理请求头参数 (header 标签) -> 映射至 OpenAPI header 参数
		if tag := f.Tag.Get("header"); tag != "" {
			c.addParam(op, tag, "header", desc, required, f.Type)
			continue
		}

		// 4. 收集正体字段 (json 标签)
		// 如果字段标记为 json:"-" 则跳过，否则即使没有标签，默认也会作为 JSON Body 的一部分
		if f.Tag.Get("json") != "-" {
			jsonFields = append(jsonFields, f)
		}
	}

	// 如果没有发现任何 Body 字段，则不生成 RequestBody
	if len(jsonFields) == 0 {
		return
	}

	// 利用反射动态构造一个匿名结构体，代表最终的 JSON 请求体结构
	schema := c.Components.Schemas.Schema(reflect.StructOf(jsonFields))
	if schema == nil {
		return
	}

	// 设置 Operation 的请求体信息
	op.RequestBody = &RequestBody{
		Content: map[string]*MediaType{
			"application/json": {Schema: schema},
		},
		Required: len(schema.Required) > 0, // 如果 Schema 中有必填项，则 RequestBody 也是必填的
	}
}

// addParam 辅助方法：向 Operation 中添加一个新的参数描述 (path, query, header 等)
func (c *API) addParam(op *Operation, name, in, desc string, required bool, t reflect.Type) {
	op.Parameters = append(op.Parameters, &Param{
		Name:        name,
		In:          in,
		Required:    required,
		Description: desc,
		Schema:      c.Schema(t), // 自动解析类型对应的 JSON Schema
	})
}

// parseResponseBody 解析处理器的返回值类型，并根据需要自动注入默认的 200 响应。
func (c *API) parseResponseBody(op *Operation, t reflect.Type) {
	// 仅当用户未在路由定义中显式通过 AddOperation 自定义 200 响应时，才执行自动注入。
	if _, ok := op.Responses["200"]; ok {
		return
	}

	// 如果处理器没有返回值，注入一个不带 Body 的 200 响应。
	if t == nil {
		op.Responses["200"] = &ResponseBody{}
		return
	}

	// 否则，解析返回类型并生成 application/json 响应
	op.Responses["200"] = &ResponseBody{
		Content: map[string]*MediaType{
			"application/json": {Schema: c.Schema(t)},
		},
	}
}

// registerOperation 将 Operation 注册到 OpenAPI 的 Paths 映射中，并执行标签同步
func (c *API) registerOperation(op *Operation, path, method string) {
	method = strings.ToUpper(method)
	// 将 Gin 风格的 :param 或 *param 转换为 OpenAPI 风格的 {param}
	p := pathRegex.ReplaceAllString(path, "{$2}")

	// 初始化路径项
	if _, ok := c.Paths[p]; !ok {
		c.Paths[p] = &PathItem{}
	}

	// 根据 HTTP 方法将 Operation 挂载到对应的路径项上
	switch item := c.Paths[p]; method {
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

	// 标签同步逻辑：确保 Operation 中使用的所有标签都在 OpenAPI 根对象的 tags 列表中声明。
	// 这有助于 UI 文档工具 (如 Swagger UI) 正确显示和分类接口。
	if c.tagMap == nil {
		c.tagMap = map[string]bool{}
	}

	for _, tag := range op.Tags {
		if !c.tagMap[tag] {
			c.tagMap[tag] = true
			c.Tags = append(c.Tags, &Tag{Name: tag}) // 发现新标签，同步到全局 tags 声明。
		}
	}
}
