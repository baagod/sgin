package main

import (
	"github.com/baagod/sgin"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	Path int    `uri:"path"`
	Head int    `head:"head"`
}

type Result struct {
	Status int    `json:"status"`
	Code   string `json:"code"`
	Data   any    `json:"data"`
}

type Login struct {
	Username string `json:"username" doc:"用户名" binding:"required,min=3"`
	Password string `json:"password" doc:"密码" binding:"required,min=6"`
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

	r.POST("/users/:id", sgin.Ho(func(c *sgin.Ctx, u Login) Login {
		return u
	}))

	_ = r.Run(":8080")

	// g := gin.Default()
	// g.GET("/", func(c *gin.Context) {
	//
	// }, func(c *gin.Context) {
	//
	// }, func(c *gin.Context) {
	//
	// })
	//
	// _ = g.Run(":8080")
}
