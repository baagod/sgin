package oa

import (
    "bytes"
    "net/http"
    "reflect"
    "regexp"
    "strings"

    "github.com/baagod/sgin/helper"
    "gopkg.in/yaml.v3"
)

const Version = "3.1.1"

var (
    pathRegex = regexp.MustCompile(`([:*])([^/]+)`)
)

type (
    AddOperation func(*Operation)
    Requirement  map[string][]string
)

type OpenAPI struct {
    OpenAPI    string               `yaml:"openapi"`
    Info       *Info                `yaml:"info"`
    Paths      map[string]*PathItem `yaml:"paths,omitempty"`
    Components *Components          `yaml:"components"`
    Security   []Requirement        `yaml:"security,omitempty"`
    Tags       []*Tag               `yaml:"tags,omitempty"`

    tagMap map[string]bool
    config Config
}

type Info struct {
    Title   string `yaml:"title"`
    Version string `yaml:"version"`
}

type Tag struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description,omitempty"`
}

type PathItem struct {
    Ref         string `yaml:"$ref,omitempty"`
    Summary     string `yaml:"summary,omitempty"`
    Description string `yaml:"description,omitempty"`

    Get     *Operation `yaml:"get,omitempty"`
    Put     *Operation `yaml:"put,omitempty"`
    Post    *Operation `yaml:"post,omitempty"`
    Delete  *Operation `yaml:"delete,omitempty"`
    Options *Operation `yaml:"options,omitempty"`
    Head    *Operation `yaml:"head,omitempty"`
    Patch   *Operation `yaml:"patch,omitempty"`
    Trace   *Operation `yaml:"trace,omitempty"`

    Parameters []*Param `yaml:"parameters,omitempty"`
}

type Operation struct {
    Summary     string               `yaml:"summary,omitempty"`
    Description string               `yaml:"description,omitempty"`
    Parameters  []*Param             `yaml:"parameters,omitempty"`
    RequestBody *RequestBody         `yaml:"requestBody,omitempty"`
    Responses   map[string]*Response `yaml:"responses,omitempty"`
    Security    []Requirement        `yaml:"security,omitempty"`
    Tags        []string             `yaml:"tags,omitempty"`
}

type Param struct {
    Ref         string  `yaml:"$ref,omitempty"`
    Name        string  `yaml:"name,omitempty"`
    In          string  `yaml:"in,omitempty"` // "query", "header", "path", "cookie"
    Required    bool    `yaml:"required,omitempty"`
    Description string  `yaml:"description,omitempty"`
    Schema      *Schema `yaml:"schema,omitempty"`
}

type RequestBody struct {
    Ref         string                `yaml:"$ref,omitempty"`
    Description string                `yaml:"description,omitempty"`
    Content     map[string]*MediaType `yaml:"content"`
    Required    bool                  `yaml:"required,omitempty"`
}

type Response struct {
    Ref         string                `yaml:"$ref,omitempty"`
    Description string                `yaml:"description,omitempty"`
    Headers     map[string]*Param     `yaml:"headers,omitempty"`
    Content     map[string]*MediaType `yaml:"content,omitempty"`
}

type MediaType struct {
    Schema *Schema `yaml:"schema,omitempty"`
}

type Components struct {
    Schemas         map[string]*Schema         `yaml:"schemas,omitempty"`
    SecuritySchemes map[string]*SecurityScheme `yaml:"securitySchemes,omitempty"`
}

// YAML 返回 YAML 格式的 OpenAPI 规范
func (oa *OpenAPI) YAML() ([]byte, error) {
    var buf bytes.Buffer
    enc := yaml.NewEncoder(&buf)
    enc.SetIndent(2)

    if err := enc.Encode(oa); err != nil {
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

func (oa *OpenAPI) Config() Config {
    return oa.config
}

func (oa *OpenAPI) Register(op *Operation, path, method string, handler any) {
    if oa == nil {
        return
    }

    if op.Responses == nil {
        op.Responses = map[string]*Response{}
    }

    // 1. 分析入参 (Request)
    t := reflect.TypeOf(handler) // type: sgin.Handler
    if t.NumIn() == 2 {          // func(ctx, input)
        oa.parseRequestParams(op, t.In(1))
    }

    // 2. 分析出参 (Response)
    var resType reflect.Type
    for i := 0; i < t.NumOut(); i++ {
        if out := t.Out(i); out.Name() != "error" {
            resType = out
            break
        }
    }

    oa.parseResponseBody(op, resType)      // 解析响应体
    oa.registerOperation(op, path, method) // 注册操作对象
}

func (oa *OpenAPI) registerOperation(op *Operation, path, method string) {
    method = strings.ToUpper(method)
    apiPath := pathRegex.ReplaceAllString(path, "{$2}")

    if _, ok := oa.Paths[apiPath]; !ok {
        oa.Paths[apiPath] = &PathItem{}
    }

    switch item := oa.Paths[apiPath]; method {
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
        if !oa.tagMap[tag] {
            oa.tagMap[tag] = true
        }
    }
}

// parseRequestParams 解析请求参数 (Path, Query, Header)
func (oa *OpenAPI) parseRequestParams(op *Operation, t reflect.Type) {
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
            oa.addParam(op, tag, "path", desc, true, f.Type)
            continue
        }

        if tag := f.Tag.Get("form"); tag != "" {
            oa.addParam(op, tag, "query", desc, required, f.Type)
            continue
        }

        if tag := f.Tag.Get("header"); tag != "" {
            oa.addParam(op, tag, "header", desc, required, f.Type)
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
    schema := oa.schemaFromType(reflect.StructOf(fields))
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

func (oa *OpenAPI) addParam(op *Operation, name, in, desc string, required bool, t reflect.Type) {
    op.Parameters = append(op.Parameters, &Param{
        Name:        name,
        In:          in,
        Required:    required,
        Description: desc,
        Schema:      oa.schemaFromType(t),
    })
}

// parseResponseBody 解析响应体
func (oa *OpenAPI) parseResponseBody(op *Operation, t reflect.Type) {
    if t == nil {
        op.Responses["200"] = &Response{}
        return
    }

    op.Responses["200"] = &Response{
        Content: map[string]*MediaType{
            "application/json": {
                Schema: oa.schemaFromType(t),
            },
        },
    }
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
    <body style="height: 100vh;">
        <elements-api
            apiDescriptionUrl="/openapi.yaml"
            router="hash"
            layout="sidebar"
        />
    </body>
</html>
`
