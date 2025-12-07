package sgin

import (
	"reflect"
	"strings"
)

// --- OpenAPI 3.2.0 基础结构 (OA 前缀避免命名冲突) ---

type OpenAPISpec struct {
	OpenAPI    string              `json:"openapi"` // e.g., "3.2.0"
	Info       OAInfo              `json:"info"`
	Paths      map[string]OAPathItem `json:"paths"`
	Components OAComponents        `json:"components"`
}

type OAInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// OAPathItem 对应一个路径下的操作集合 (Method -> Operation)
type OAPathItem map[string]OAOperation

type OAOperation struct {
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Parameters  []OAParameter         `json:"parameters,omitempty"`
	RequestBody *OARequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]OAResponse `json:"responses"`
	Tags        []string              `json:"tags,omitempty"`
}

type OAParameter struct {
	Name        string    `json:"name"`
	In          string    `json:"in"` // "query", "header", "path", "cookie"
	Required    bool      `json:"required"`
	Description string    `json:"description,omitempty"`
	Schema      *OASchema `json:"schema,omitempty"`
}

type OARequestBody struct {
	Description string                 `json:"description,omitempty"`
	Content     map[string]OAMediaType `json:"content"`
	Required    bool                   `json:"required"`
}

type OAResponse struct {
	Description string                 `json:"description"`
	Content     map[string]OAMediaType `json:"content,omitempty"`
}

type OAMediaType struct {
	Schema *OASchema `json:"schema"`
}

type OAComponents struct {
	Schemas map[string]*OASchema `json:"schemas,omitempty"`
}

type OASchema struct {
	Type        string               `json:"type,omitempty"`
	Format      string               `json:"format,omitempty"`
	Properties  map[string]*OASchema `json:"properties,omitempty"`
	Items       *OASchema            `json:"items,omitempty"` // For arrays
	Required    []string             `json:"required,omitempty"`
	Description string               `json:"description,omitempty"`
	Example     any                  `json:"example,omitempty"`
	Ref         string               `json:"$ref,omitempty"`
}

// 全局 OpenAPI 实例
var globalSpec = &OpenAPISpec{
	OpenAPI: "3.2.0",
	Info: OAInfo{
		Title:   "Sgin API",
		Version: "1.0.0",
	},
	Paths: make(map[string]OAPathItem),
	Components: OAComponents{
		Schemas: make(map[string]*OASchema),
	},
}

// AnalyzeAndRegister 分析 Handler 并注册到 OpenAPI
func AnalyzeAndRegister(path string, method string, handler any) {
	t := reflect.TypeOf(handler)
	if t.Kind() != reflect.Func {
		return
	}

	var reqType, resType reflect.Type
	
	// 分析入参
	if t.NumIn() == 2 {
		reqType = t.In(1)
	}
	
	// 分析出参
	if t.NumOut() > 0 {
		for i := 0; i < t.NumOut(); i++ {
			out := t.Out(i)
			if out.Name() != "error" && out.Kind() != reflect.Int {
				resType = out
				break
			}
		}
	}
	
	registerOperation(path, method, t, reqType, resType)
}

// registerOperation 注册操作元数据
func registerOperation(path string, method string, handlerType reflect.Type, reqType reflect.Type, resType reflect.Type) {
	if globalSpec.Paths == nil {
		globalSpec.Paths = make(map[string]OAPathItem)
	}

	openAPIPath := convertPath(path)

	if _, ok := globalSpec.Paths[openAPIPath]; !ok {
		globalSpec.Paths[openAPIPath] = make(OAPathItem)
	}

	op := OAOperation{
		Responses: make(map[string]OAResponse),
	}

	// 1. 解析 Request
	if reqType != nil {
		op.parseRequest(reqType)
	}

	// 2. 解析 Response
	op.parseResponse(resType)

	// 3. 注册
	globalSpec.Paths[openAPIPath][strings.ToLower(method)] = op
}

func convertPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		} else if strings.HasPrefix(part, "*") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

// parseRequest 解析请求
func (op *OAOperation) parseRequest(t reflect.Type) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		desc := field.Tag.Get("doc")

		if tag := field.Tag.Get("uri"); tag != "" {
			op.Parameters = append(op.Parameters, OAParameter{
				Name:        tag,
				In:          "path",
				Required:    true,
				Description: desc,
				Schema:      typeToSchema(field.Type),
			})
		}

		if tag := field.Tag.Get("form"); tag != "" {
			op.Parameters = append(op.Parameters, OAParameter{
				Name:        tag,
				In:          "query",
				Required:    strings.Contains(field.Tag.Get("binding"), "required"),
				Description: desc,
				Schema:      typeToSchema(field.Type),
			})
		}

		if tag := field.Tag.Get("header"); tag != "" {
			op.Parameters = append(op.Parameters, OAParameter{
				Name:        tag,
				In:          "header",
				Required:    strings.Contains(field.Tag.Get("binding"), "required"),
				Description: desc,
				Schema:      typeToSchema(field.Type),
			})
		}
	}
	
	hasJSON := false
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Tag.Get("json") != "" {
			hasJSON = true
			break
		}
	}
	
	if hasJSON {
		schemaName := t.Name()
		if schemaName == "" {
			schemaName = "Request"
		}
		
		if globalSpec.Components.Schemas == nil {
			globalSpec.Components.Schemas = make(map[string]*OASchema)
		}
		globalSpec.Components.Schemas[schemaName] = structToSchema(t)
		
		op.RequestBody = &OARequestBody{
			Content: map[string]OAMediaType{
				"application/json": {
					Schema: &OASchema{Ref: "#/components/schemas/" + schemaName},
				},
			},
		}
	}
}

// parseResponse 解析响应
func (op *OAOperation) parseResponse(t reflect.Type) {
	if t == nil {
		op.Responses["200"] = OAResponse{Description: "OK"}
		return
	}
	
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	schemaName := t.Name()
	if schemaName == "" {
		schemaName = "Response"
	}
	
	if globalSpec.Components.Schemas == nil {
		globalSpec.Components.Schemas = make(map[string]*OASchema)
	}
	globalSpec.Components.Schemas[schemaName] = structToSchema(t)
	
	op.Responses["200"] = OAResponse{
		Description: "OK",
		Content: map[string]OAMediaType{
			"application/json": {
				Schema: &OASchema{Ref: "#/components/schemas/" + schemaName},
			},
		},
	}
}

func typeToSchema(t reflect.Type) *OASchema {
	switch t.Kind() {
	case reflect.String:
		return &OASchema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &OASchema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &OASchema{Type: "number"}
	case reflect.Bool:
		return &OASchema{Type: "boolean"}
	case reflect.Struct:
		return &OASchema{Type: "object"}
	default:
		return &OASchema{Type: "string"}
	}
}

func structToSchema(t reflect.Type) *OASchema {
	schema := &OASchema{
		Type:       "object",
		Properties: make(map[string]*OASchema),
	}
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]
		
		propSchema := typeToSchema(field.Type)
		propSchema.Description = field.Tag.Get("doc")
		
		schema.Properties[name] = propSchema
	}
	return schema
}

const swaggerHTML = `
<!doctype html>
<html>
  <head>
    <title>API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/openapi.json"
      data-configuration="{
        theme: 'default', // 可以是 'default', 'alternate', 'purple', 'solarized'
        layout: 'modern', // 可以是 'modern', 'split'
        // 其他更多配置项...
      }"
      src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"
    ></script>
  </body>
</html>
`