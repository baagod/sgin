package sgin

import (
	"github.com/gin-gonic/gin/binding"
	"net/http"
)

var (
	Uri    = &uri{}
	Form   = &binding.Form
	JSON   = &binding.JSON
	XML    = &binding.XML
	TOML   = &binding.TOML
	YAML   = &binding.YAML
	Header = &binding.Header
)

type uri struct{}

func (uri) Name() string {
	return "uri"
}

func (uri) BindUri(m map[string][]string, obj any) error {
	return binding.Uri.BindUri(m, obj)
}

func (uri) Bind(*http.Request, any) error {
	panic("please use BindUri()")
}

type Binding struct {
	Uri      binding.BindingUri
	Bindings []binding.Binding
}

func Bind(bb ...binding.Binding) (opt *Binding) {
	opt = &Binding{Bindings: bb}
	for i, x := range bb {
		if x.Name() == "uri" {
			opt.Uri = x.(binding.BindingUri)
			opt.Bindings = append(bb[:i], bb[i+1:]...)
			return
		}
	}
	return
}
