package main

import (
	"github.com/baagod/sgin/v2"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	Path int    `uri:"path"`
	Head int    `head:"head"`
}

func main() {
	r := sgin.New(sgin.Config{
		Mode:    gin.DebugMode,
		OpenAPI: sgin.NewAPI(),
		Locales: []language.Tag{
			language.Chinese, // 默认语言（第一个）
			language.English, // 备用语言
		},
	})

	r.POST("/users/:id", sgin.Ho(func(ctx *sgin.Ctx, _ struct{}) (r *sgin.Result) {
		return r.SetCode("").SetStatus("")
	}))

	_ = r.Run(":8080")
}
