package sgin

import (
    "reflect"
    "strings"
)

const OpenAPIVersion = "3.1"

// --- OpenAPI 基础结构 ---

type OASecurityRequirement map[string][]string // e.g., {"bearerAuth": []}

type OpenAPISpec struct {
    OpenAPI    string                  `json:"openapi"`
    Info       OAInfo                  `json:"info"`
    Paths      map[string]OAPathItem   `json:"paths"`
    Components OAComponents            `json:"components"`
    Security   []OASecurityRequirement `json:"security,omitempty"` // 全局安全配置
    Servers    []map[string]any        `json:"servers,omitempty"`
}

type OAInfo struct {
    Title   string `json:"title"`
    Version string `json:"version"`
}

// OAPathItem 对应一个路径下的操作集合 (Method -> Operation)
type OAPathItem map[string]OAOperation

type OAOperation struct {
    Summary     string                  `json:"summary,omitempty"`
    Description string                  `json:"description,omitempty"`
    Parameters  []OAParameter           `json:"parameters,omitempty"`
    RequestBody *OARequestBody          `json:"requestBody,omitempty"`
    Responses   map[string]OAResponse   `json:"responses"`
    Security    []OASecurityRequirement `json:"security,omitempty"`
    Tags        []string                `json:"tags,omitempty"`
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
    Schemas         map[string]*OASchema        `json:"schemas,omitempty"`
    SecuritySchemes map[string]OASecurityScheme `json:"securitySchemes,omitempty"`
}

type OASecurityScheme struct {
    Type         string `json:"type"`                   // "http", "apiKey", "oauth2"
    Scheme       string `json:"scheme,omitempty"`       // "bearer" (for HTTP)
    BearerFormat string `json:"bearerFormat,omitempty"` // "JWT" (for bearer)
    Name         string `json:"name,omitempty"`         // Header name for apiKey
    In           string `json:"in,omitempty"`           // "header" for apiKey
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
    OpenAPI: OpenAPIVersion,
    Info: OAInfo{
        Title:   "Sgin API",
        Version: "1.0.0",
    },
    Paths: make(map[string]OAPathItem),
    Components: OAComponents{
        Schemas: make(map[string]*OASchema),
        SecuritySchemes: map[string]OASecurityScheme{
            "bearerAuth": {
                Type:         "http",
                Scheme:       "bearer",
                BearerFormat: "JWT Bearer token authentication",
            },
        },
    },
    Security: []OASecurityRequirement{
        {"bearerAuth": {}},
    },
}

// AnalyzeAndRegister 分析 Handler 并注册到 OpenAPI
func AnalyzeAndRegister(path string, method string, handler Handler, security []OASecurityRequirement) {
    t := reflect.TypeOf(handler)
    if t.Kind() != reflect.Func {
        return
    }

    if len(security) == 0 {
        security = make([]OASecurityRequirement, 1)
    }
    op := OAOperation{Responses: map[string]OAResponse{}, Security: security}

    // 1. 分析入参 (Request)
    // 假设第二个参数是请求结构体 func(c *Ctx, req *UserReq)
    if t.NumIn() == 2 {
        reqType := t.In(1)
        parseRequestParams(&op, reqType)
    }

    // 2. 分析出参 (Response)
    // 假设第一个返回值是响应结构体 func(...) (UserResp, error)
    var resType reflect.Type
    for i := 0; i < t.NumOut(); i++ {
        out := t.Out(i)
        // 排除 error 和 int (通常是状态码)
        if out.Name() != "error" && out.Kind() != reflect.Int {
            resType = out
            break
        }
    }
    parseResponseBody(&op, resType)

    // 3. 注册到全局 Spec
    registerOperation(path, method, op)
}

func registerOperation(path string, method string, op OAOperation) {
    if globalSpec.Paths == nil {
        globalSpec.Paths = make(map[string]OAPathItem)
    }

    openAPIPath := convertPath(path)
    if _, ok := globalSpec.Paths[openAPIPath]; !ok {
        globalSpec.Paths[openAPIPath] = make(OAPathItem)
    }
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

// parseRequestParams 解析请求参数 (Path, Query, Header)
func parseRequestParams(op *OAOperation, t reflect.Type) {
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    if t.Kind() != reflect.Struct {
        return
    }

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        desc := field.Tag.Get("doc")

        // 提取 Tag
        if tag := field.Tag.Get("uri"); tag != "" {
            addParam(op, tag, "path", true, desc, field.Type)
        }
        if tag := field.Tag.Get("form"); tag != "" {
            required := strings.Contains(field.Tag.Get("binding"), "required")
            addParam(op, tag, "query", required, desc, field.Type)
        }
        if tag := field.Tag.Get("header"); tag != "" {
            required := strings.Contains(field.Tag.Get("binding"), "required")
            addParam(op, tag, "header", required, desc, field.Type)
        }

        // TODO: 处理 Body (JSON) - 可以在这里检测 json tag 并生成 RequestBody
    }
}

func addParam(op *OAOperation, name, in string, required bool, desc string, t reflect.Type) {
    op.Parameters = append(op.Parameters, OAParameter{
        Name:        name,
        In:          in,
        Required:    required,
        Description: desc,
        Schema:      getSchema(t),
    })
}

// parseResponseBody 解析响应体
func parseResponseBody(op *OAOperation, t reflect.Type) {
    if t == nil {
        op.Responses["200"] = OAResponse{Description: "OK"}
        return
    }

    op.Responses["200"] = OAResponse{
        Description: "OK",
        Content: map[string]OAMediaType{
            "application/json": {
                Schema: getSchema(t),
            },
        },
    }
}

// getSchema 递归生成 Schema，支持基础类型、切片、Map 和结构体引用
func getSchema(t reflect.Type) *OASchema {
    if t == nil {
        return &OASchema{Type: "string"} // Fallback
    }
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }

    switch t.Kind() {
    case reflect.Bool:
        return &OASchema{Type: "boolean"}
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
        reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        return &OASchema{Type: "integer"}
    case reflect.Float32, reflect.Float64:
        return &OASchema{Type: "number"}
    case reflect.String:
        return &OASchema{Type: "string"}
    case reflect.Slice, reflect.Array:
        return &OASchema{
            Type:  "array",
            Items: getSchema(t.Elem()),
        }
    case reflect.Map:
        return &OASchema{
            Type: "object",
            Properties: map[string]*OASchema{
                "additionalProperties": getSchema(t.Elem()),
            },
        }
    case reflect.Struct:
        // 如果是时间类型，特殊处理
        if t.Name() == "Time" && t.PkgPath() == "time" {
            return &OASchema{Type: "string", Format: "date-time"}
        }
        return registerStructSchema(t)
    default:
        return &OASchema{Type: "string"} // Interface or other
    }
}

// registerStructSchema 将结构体注册到 Components 并返回 $ref
func registerStructSchema(t reflect.Type) *OASchema {
    name := t.Name()
    if name == "" {
        name = "AnonymousStruct" // 匿名结构体无法引用，只能内联（此处简化处理）
        // 实际上应该生成内联 Schema，或者生成一个随机名字
        // 简单起见，这里先内联
        return generateInlineStructSchema(t)
    }

    // 检查是否已注册
    if globalSpec.Components.Schemas == nil {
        globalSpec.Components.Schemas = make(map[string]*OASchema)
    }
    if _, ok := globalSpec.Components.Schemas[name]; ok {
        return &OASchema{Ref: "#/components/schemas/" + name}
    }

    // 先占位，防止递归死循环
    globalSpec.Components.Schemas[name] = &OASchema{}

    // 生成 Schema
    schema := generateInlineStructSchema(t)
    globalSpec.Components.Schemas[name] = schema

    return &OASchema{Ref: "#/components/schemas/" + name}
}

func generateInlineStructSchema(t reflect.Type) *OASchema {
    schema := &OASchema{
        Type:       "object",
        Properties: make(map[string]*OASchema),
    }

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        // 处理 JSON Tag
        jsonTag := field.Tag.Get("json")
        if jsonTag == "-" {
            continue
        }
        propName := field.Name
        if jsonTag != "" {
            parts := strings.Split(jsonTag, ",")
            propName = parts[0]
        }

        propSchema := getSchema(field.Type)
        propSchema.Description = field.Tag.Get("doc")

        // 处理 required
        binding := field.Tag.Get("binding")
        if strings.Contains(binding, "required") {
            schema.Required = append(schema.Required, propName)
        }

        schema.Properties[propName] = propSchema
    }
    return schema
}

const swaggerHTML = `
<!doctype html>
<html lang="zh">
    <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
        <title>API References</title>
        <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
        <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css">
    </head>
    <body>
        <elements-api
            apiDescriptionUrl="openapi.json"
            router="hash"
        />
    </body>
</html>
`
