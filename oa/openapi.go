package oa

import (
	"bytes"
	"regexp"

	"gopkg.in/yaml.v3"
)

const Version = "3.1.2"

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
	Schemas         *Registry                  `yaml:"schemas,omitempty"`
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
