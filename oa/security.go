package oa

type SecurityScheme struct {
    Type         string      `yaml:"type"` // "http", "apiKey", "oauth2"
    Description  string      `yaml:"description,omitempty"`
    Name         string      `yaml:"name,omitempty"`         // Header name for apiKey
    In           string      `yaml:"in,omitempty"`           // "header" for apiKey
    Scheme       string      `yaml:"scheme,omitempty"`       // "bearer" (for HTTP)
    BearerFormat string      `yaml:"bearerFormat,omitempty"` // "JWT" (for bearer)
    Flows        *OAuthFlows `yaml:"flows,omitempty"`
}

type OAuthFlows struct {
    ClientCredentials *OAuthFlow `yaml:"clientCredentials,omitempty"`
    AuthorizationCode *OAuthFlow `yaml:"authorizationCode,omitempty"`
}

type OAuthFlow struct {
    AuthorizationUrl string            `yaml:"authorizationUrl,omitempty"` // 鉴权地址
    TokenUrl         string            `yaml:"tokenUrl,omitempty"`         // 令牌地址
    RefreshUrl       string            `yaml:"refreshUrl,omitempty"`       // 刷新地址 (可选)
    Scopes           map[string]string `yaml:"scopes"`                     // 作用域定义
}
