package sgin

import (
	"github.com/gin-gonic/gin/binding"
	"net/http"
)

var Uri = uri{}

type uri struct{}

func (uri) Name() string {
	return "uri"
}

func (uri) BindUri(m map[string][]string, obj any) error {
	return binding.Uri.BindUri(m, obj)
}

func (uri) Bind(*http.Request, any) error {
	return nil
}

type BindingOption struct {
	Uri      binding.BindingUri
	Bindings []binding.Binding
}

func Bindings(opt ...binding.Binding) *BindingOption {
	opts := &BindingOption{Bindings: opt}
	for i, x := range opt {
		if x.Name() == "uri" {
			opts.Uri = x.(binding.BindingUri)
			opts.Bindings = append(opt[:i], opt[i+1:]...)
			continue
		}
	}
	return opts
}
