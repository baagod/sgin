package sgin

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/baagod/sgin/helper"
)

// isFileType 检查类型是否为文件上传类型 (*multipart.FileHeader 或 []*multipart.FileHeader)
func isFileType(t reflect.Type) bool {
	t = helper.Deref(t)
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = helper.Deref(t.Elem())
	}
	return t == fileHeaderType
}

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

func (a *API) Schema(t reflect.Type, hint ...string) *Schema {
	return a.Components.Schemas.Schema(t, hint...)
}

func (a *API) Struct(t reflect.Type, hint ...string) *Schema {
	return a.Components.Schemas.Struct(t, hint...)
}

func (a *API) Field(f reflect.StructField, hint string) (s *Schema) {
	return a.Components.Schemas.Field(f, hint)
}

// Register 将处理器的元数据 (输入/输出类型) 注册到指定的 Operation 中
func (a *API) Register(op *Operation, path, method string, arg *HandleArg) {
	if arg == nil || op.Hidden {
		return
	}

	// 确保响应对象已初始化，避免后续注入时出现 nil map
	if op.Responses == nil {
		op.Responses = map[string]*ResponseBody{}
	}

	a.parseRequestParams(op, arg.In)      // 解析结构体标签并映射为请求参数或 RequestBody
	a.parseResponseBody(op, arg.Out)      // 解析返回类型并映射为 ResponseBody
	a.registerOperation(op, path, method) // 将配置好的 Operation 绑定到 OpenAPI 路径树中
}

// parseRequestParams 解析输入标签 (uri, form, header, json) 并映射为 OpenAPI 的参数或请求体
func (a *API) parseRequestParams(op *Operation, t reflect.Type) {
	t = helper.Deref(t)
	if t.Kind() != reflect.Struct {
		return
	}

	var body []reflect.StructField // 用于收集映射到 RequestBody 的字段
	mime := MIMEJSON               // 默认媒体类型

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		desc := f.Tag.Get("doc")                                       // 获取描述
		required := strings.Contains(f.Tag.Get("binding"), "required") // 检查是否必填

		// 1. 处理路径参数 (uri 标签) -> 映射至 OpenAPI path 参数
		if tag := f.Tag.Get("uri"); tag != "" {
			a.addParam(op, tag, "path", desc, true, f.Type)
			continue
		}

		// 2. 处理查询参数 (form 标签)
		// form 标签始终映射为 Query 参数，除非它是文件类型。
		if tag := f.Tag.Get("form"); tag != "" {
			if isFileType(f.Type) {
				body = append(body, f)
				mime = MIMEMultipartForm
			} else {
				a.addParam(op, tag, "query", desc, required, f.Type)
			}
			continue
		}

		// 3. 处理请求头参数 (header 标签) -> 映射至 OpenAPI header 参数
		if tag := f.Tag.Get("header"); tag != "" {
			a.addParam(op, tag, "header", desc, required, f.Type)
			continue
		}

		// 4. 收集正体字段 (json 标签)
		if tag := f.Tag.Get("json"); tag != "-" {
			if isFileType(f.Type) && mime != MIMEMultipartForm {
				mime = MIMEMultipartForm
			}
			body = append(body, f)
		}
	}

	if len(body) == 0 {
		return
	}

	// 手动构建 RequestBody 的 Properties，
	// 以保留字段级别的元数据 (如 format, doc, default, enum)。
	var required []string
	props := map[string]*Schema{}

	for _, f := range body {
		name := f.Name
		if mime == MIMEMultipartForm {
			name = f.Tag.Get("form")
		} else {
			if name = strings.Split(f.Tag.Get("json"), ",")[0]; name == "-" {
				name = f.Tag.Get("form")
			}
		}

		if fs := a.Field(f, t.Name()+f.Name); fs != nil {
			props[name] = fs
			if strings.Contains(f.Tag.Get("binding"), "required") {
				required = append(required, name)
			}
		}
	}

	// 注入到 Operation 的 RequestBody 中
	op.RequestBody = &RequestBody{
		Content: map[string]*MediaType{
			mime: {
				Schema: &Schema{Type: TypeObject, Properties: props, Required: required},
			},
		},
		Required: len(required) > 0,
	}
}

// addParam 辅助方法：向 Operation 中添加一个新的参数描述 (path, query, header 等)
func (a *API) addParam(op *Operation, name, in, desc string, required bool, t reflect.Type) {
	op.Parameters = append(op.Parameters, &Param{
		Name:        name,
		In:          in,
		Required:    required,
		Description: desc,
		Schema:      a.Schema(t), // 自动解析类型对应的 JSON Schema
	})
}

// parseResponseBody 解析处理器的返回值类型，并根据需要自动注入默认的 200 响应。
func (a *API) parseResponseBody(op *Operation, t reflect.Type) {
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
		Content: map[string]*MediaType{MIMEJSON: {Schema: a.Schema(t)}},
	}
}

// registerOperation 将 op 注册到 OpenAPI 的 Paths 映射并执行标签同步
func (a *API) registerOperation(op *Operation, path, method string) {
	method = strings.ToUpper(method)
	// 将 Gin 风格的 :param 或 *param 转换为 OpenAPI 风格的 {param}
	p := pathRegex.ReplaceAllString(path, "{$2}")

	// 初始化路径项
	if _, ok := a.Paths[p]; !ok {
		a.Paths[p] = &PathItem{}
	}

	// 根据 HTTP 方法将 Operation 挂载到对应的路径项上
	switch item := a.Paths[p]; method {
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

	// 标签同步逻辑：确保 Operation 中使用的所有标签都在 OpenAPI 根对象的 tags 列表中
	if a.tagMap == nil {
		a.tagMap = map[string]bool{}
	}

	for _, tag := range op.Tags {
		if !a.tagMap[tag] {
			a.tagMap[tag] = true
			a.Tags = append(a.Tags, &Tag{Name: tag}) // 发现新标签，同步到全局 tags。
		}
	}
}
