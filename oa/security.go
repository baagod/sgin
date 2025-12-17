package oa

type SecurityScheme struct {
    Type         string `yaml:"type"` // "http", "apiKey", "oauth2"
    Description  string `yaml:"description,omitempty"`
    Name         string `yaml:"name,omitempty"`         // Header name for apiKey
    In           string `yaml:"in,omitempty"`           // "header" for apiKey
    Scheme       string `yaml:"scheme,omitempty"`       // "bearer" (for HTTP)
    BearerFormat string `yaml:"bearerFormat,omitempty"` // "JWT" (for bearer)
}
