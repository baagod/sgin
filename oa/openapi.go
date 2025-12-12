package oa

import (
    "bytes"
    "reflect"
    "strings"

    "gopkg.in/yaml.v3"
)

const Version = "3.1.1"

type (
    AddOperation func(*Operation)
    Requirement  map[string][]string
    PathItem     map[string]Operation
)

type OpenAPI struct {
    OpenAPI    string              `yaml:"openapi"`
    Info       Info                `yaml:"info"`
    Servers    []Server            `yaml:"servers,omitempty"` // todo: ...
    Paths      map[string]PathItem `yaml:"paths"`
    Components Components          `yaml:"components"`
    Security   []Requirement       `yaml:"security,omitempty"`
    Tags       []Tag               `yaml:"tags,omitempty"`
}

type Info struct {
    Title   string `yaml:"title"`
    Version string `yaml:"version"`
}

type Tag struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description,omitempty"`
}

type Server struct {
    URL         string                    `yaml:"url"`
    Description string                    `yaml:"description,omitempty"`
    Variables   map[string]ServerVariable `yaml:"variables,omitempty"`
}

type ServerVariable struct {
    Enum        []string `yaml:"enum,omitempty"`
    Default     string   `yaml:"default"`
    Description string   `yaml:"description,omitempty"`
}

type Operation struct {
    Summary     string              `yaml:"summary,omitempty"`
    Description string              `yaml:"description,omitempty"`
    Parameters  []Param             `yaml:"parameters,omitempty"`
    RequestBody *RequestBody        `yaml:"requestBody,omitempty"`
    Responses   map[string]Response `yaml:"responses"`
    Security    []Requirement       `yaml:"security,omitempty"`
    Tags        []string            `yaml:"tags,omitempty"`
}

type Param struct {
    Name        string  `yaml:"name"`
    In          string  `yaml:"in"` // "query", "header", "path", "cookie"
    Required    bool    `yaml:"required"`
    Description string  `yaml:"description,omitempty"`
    Schema      *Schema `yaml:"schema,omitempty"`
}

type RequestBody struct {
    Description string               `yaml:"description,omitempty"`
    Content     map[string]MediaType `yaml:"content"`
    Required    bool                 `yaml:"required"`
}

type Response struct {
    Description string               `yaml:"description"`
    Content     map[string]MediaType `yaml:"content,omitempty"`
}

type MediaType struct {
    Schema *Schema `yaml:"schema"`
}

type Components struct {
    Schemas         map[string]*Schema        `yaml:"schemas,omitempty"`
    SecuritySchemes map[string]SecurityScheme `yaml:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
    Type         string `yaml:"type"` // "http", "apiKey", "oauth2"
    Description  string `yaml:"description,omitempty"`
    Name         string `yaml:"name,omitempty"`         // Header name for apiKey
    In           string `yaml:"in,omitempty"`           // "header" for apiKey
    Scheme       string `yaml:"scheme,omitempty"`       // "bearer" (for HTTP)
    BearerFormat string `yaml:"bearerFormat,omitempty"` // "JWT" (for bearer)
    // flows
    OpenIdConnectUrl  string `yaml:"openIdConnectUrl,omitempty"`
    Oauth2MetadataUrl string `yaml:"oauth2MetadataUrl,omitempty"`
}

type Schema struct {
    Type                 string             `yaml:"type,omitempty"`
    Format               string             `yaml:"format,omitempty"`
    Properties           map[string]*Schema `yaml:"properties,omitempty"`
    AdditionalProperties any                `yaml:"additionalProperties,omitempty"` // Schema or bool
    Items                *Schema            `yaml:"items,omitempty"`                // For arrays
    Required             []string           `yaml:"required,omitempty"`
    Description          string             `yaml:"description,omitempty"`
    Example              any                `yaml:"example,omitempty"`
    Ref                  string             `yaml:"$ref,omitempty"`
    Nullable             bool               `yaml:"nullable,omitempty"`
}

// YAML 返回 YAML 格式的 OpenAPI 规范
func (o *OpenAPI) YAML() ([]byte, error) {
    var buf bytes.Buffer
    enc := yaml.NewEncoder(&buf)
    enc.SetIndent(2)

    if err := enc.Encode(o); err != nil {
        return nil, err
    }

    _ = enc.Close()
    return buf.Bytes(), nil
}

// Clone 返回一份深度的 Operation 副本
func (o *Operation) Clone() *Operation {
    if o == nil {
        return nil
    }
    var clone Operation
    if data, err := yaml.Marshal(o); err == nil {
        _ = yaml.Unmarshal(data, &clone)
    }
    return &clone
}

var ApiSpec = &OpenAPI{
    OpenAPI: Version,
    Info: Info{
        Title:   "Sgin API",
        Version: "1.0.0",
    },
    Paths: map[string]PathItem{},
    Components: Components{
        Schemas: map[string]*Schema{},
        SecuritySchemes: map[string]SecurityScheme{
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
            "oauth2": {
                Type: "oauth2",
            },
        },
    },
    // Security: []Requirement{{"bearerAuth": {}}},
}

// Register 分析 Handler 并注册到 OpenAPI
// 它现在接收一个已经组装好的 *Operation 对象，以及真实的 handler 函数。
func Register(path, method string, handler any, op *Operation) {
    t := reflect.TypeOf(handler)
    if t.Kind() != reflect.Func {
        return
    }

    // 1. 分析入参 (Request)
    // 假设第二个参数是请求结构体 func(c *Ctx, req *UserReq)
    if t.NumIn() == 2 {
        reqType := t.In(1)
        parseRequestParams(op, reqType)
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

    parseResponseBody(op, resType)

    // 3. 注册到全局 ApiSpec
    registerOperation(path, method, op)
}

func registerOperation(path string, method string, op *Operation) {
    if ApiSpec.Paths == nil {
        ApiSpec.Paths = make(map[string]PathItem)
    }

    openAPIPath := convertPath(path)
    if _, ok := ApiSpec.Paths[openAPIPath]; !ok {
        ApiSpec.Paths[openAPIPath] = make(PathItem)
    }
    ApiSpec.Paths[openAPIPath][strings.ToLower(method)] = *op // 注册 Operation 结构体

    // 将标签添加到全局列表 (去重)
    for _, tagName := range op.Tags { // 从 op 中获取 tags
        found := false
        for _, existingTag := range ApiSpec.Tags {
            if existingTag.Name == tagName {
                found = true
                break
            }
        }
        if !found {
            ApiSpec.Tags = append(ApiSpec.Tags, Tag{Name: tagName})
        }
    }
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
func parseRequestParams(op *Operation, t reflect.Type) {
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }

    if t.Kind() != reflect.Struct {
        return
    }

    bodySchema := &Schema{Type: "object", Properties: map[string]*Schema{}}

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        desc := field.Tag.Get("doc")

        // 提取 Tag
        if tag := field.Tag.Get("uri"); tag != "" {
            addParam(op, tag, "path", true, desc, field.Type)
            continue
        }

        if tag := field.Tag.Get("form"); tag != "" {
            required := strings.Contains(field.Tag.Get("binding"), "required")
            addParam(op, tag, "query", required, desc, field.Type)
            continue
        }

        if tag := field.Tag.Get("header"); tag != "" {
            required := strings.Contains(field.Tag.Get("binding"), "required")
            addParam(op, tag, "header", required, desc, field.Type)
            continue
        }

        // 处理 Body (JSON) - 如果没有被其他标签捕获，则视为 JSON Body 字段
        jsonTag := field.Tag.Get("json")
        if jsonTag == "-" {
            continue // 显式忽略
        }

        propName := field.Name
        if jsonTag != "" {
            propName = strings.Split(jsonTag, ",")[0]
        }

        // 确保字段 Schema 不为空
        if propSchema := getSchema(field.Type); propSchema != nil {
            propSchema.Description = desc
            bodySchema.Properties[propName] = propSchema

            if strings.Contains(field.Tag.Get("binding"), "required") {
                bodySchema.Required = append(bodySchema.Required, propName)
            }
        }
    }

    // 如果 bodySchema 中有任何属性，才将其添加到 RequestBody
    if len(bodySchema.Properties) > 0 {
        op.RequestBody = &RequestBody{
            Content: map[string]MediaType{
                "application/json": {Schema: bodySchema},
            },
        }
    }
}

func addParam(op *Operation, name, in string, required bool, desc string, t reflect.Type) {
    op.Parameters = append(op.Parameters, Param{
        Name:        name,
        In:          in,
        Required:    required,
        Description: desc,
        Schema:      getSchema(t),
    })
}

// parseResponseBody 解析响应体
func parseResponseBody(op *Operation, t reflect.Type) {
    if t == nil {
        op.Responses["200"] = Response{Description: "OK"}
        return
    }

    op.Responses["200"] = Response{
        Description: "OK",
        Content: map[string]MediaType{
            "application/json": {
                Schema: getSchema(t),
            },
        },
    }
}

// getSchema 递归生成 Schema，支持基础类型、切片、Map 和结构体引用
func getSchema(t reflect.Type) *Schema {
    if t == nil {
        return nil
    }

    isPointer := t.Kind() == reflect.Ptr
    if isPointer {
        t = t.Elem()
    }

    s := &Schema{Nullable: isPointer}

    switch t.Kind() {
    case reflect.Bool:
        s.Type = "boolean"
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
        reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        s.Type = "integer"
    case reflect.Float32:
        s.Type = "number"
        s.Format = "float"
    case reflect.Float64:
        s.Type = "number"
        s.Format = "double"
    case reflect.String:
        s.Type = "string"
    case reflect.Slice, reflect.Array:
        s.Type = "array"
        s.Items = getSchema(t.Elem())
    case reflect.Map:
        s.Type = "object"
        s.AdditionalProperties = getSchema(t.Elem())
    case reflect.Struct:
        // 如果是时间类型，特殊处理
        if t.Name() == "Time" && t.PkgPath() == "time" {
            s.Type, s.Format = "string", "date-time"
            return s
        }
        return registerStructSchema(t)
    case reflect.Interface:
        // Interfaces mean any object.
    default:
        return nil // Ignore unsupported types
    }

    return s
}

// registerStructSchema 将结构体注册到 Components 并返回 $ref
func registerStructSchema(t reflect.Type) *Schema {
    name := t.Name()
    if name == "" {
        name = "AnonymousStruct" // 匿名结构体无法引用，只能内联（此处简化处理）
        // 实际上应该生成内联 Schema，或者生成一个随机名字
        // 简单起见，这里先内联
        return generateInlineStructSchema(t)
    }

    // 检查是否已注册
    if ApiSpec.Components.Schemas == nil {
        ApiSpec.Components.Schemas = map[string]*Schema{}
    }
    if _, ok := ApiSpec.Components.Schemas[name]; ok {
        return &Schema{Ref: "#/components/schemas/" + name}
    }

    // 先占位，防止递归死循环
    ApiSpec.Components.Schemas[name] = &Schema{}

    // 生成 Schema
    schema := generateInlineStructSchema(t)
    ApiSpec.Components.Schemas[name] = schema

    return &Schema{Ref: "#/components/schemas/" + name}
}

func generateInlineStructSchema(t reflect.Type) *Schema {
    schema := &Schema{
        Type:       "object",
        Properties: map[string]*Schema{},
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
        if propSchema == nil {
            continue
        }
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

const DocsHTML = `
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
            apiDescriptionUrl="/openapi.yaml"
            router="hash"
        />
    </body>
</html>
`
