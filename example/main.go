package main

import (
	"github.com/baagod/sgin/v2"
	"github.com/gin-contrib/cors"
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
		Cors: func(c *cors.Config) {
			c.AllowCredentials = true
			c.AllowAllOrigins = true
		},
	})

	r.GET("/users/:id", sgin.Ho(func(c *sgin.Ctx, _ struct{}) (r *sgin.Result) {
		return r.SetCode("").SetStatus("")
	}))

	_ = r.Run(":8080")
}
