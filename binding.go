package sgin

import (
	"github.com/gin-gonic/gin/binding"
	"net/http"
)

var (
	Uri    = _uri{}
	Form   = binding.Form
	JSON   = binding.JSON
	XML    = binding.XML
	TOML   = binding.TOML
	YAML   = binding.YAML
	Header = binding.Header
)

type _uri struct{}

func (_uri) Name() string {
	return "uri"
}

func (_uri) BindUri(m map[string][]string, obj any) error {
	return binding.Uri.BindUri(m, obj)
}

func (_uri) Bind(*http.Request, any) error {
	panic("please use BindUri()")
}
