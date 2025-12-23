package main

import (
	"fmt"

	"github.com/baagod/sgin"
	"github.com/baagod/sgin/oa"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

type User struct {
	ID   int    `uri:"id" binding:"required"`
	Name string `form:"name" binding:"required" doc:"姓名"`
	Age  int    `form:"age" binding:"required,gt=18" doc:"年龄"`
}

type Response struct {
	Token string `json:"authorization"`
}

func main() {
	r := sgin.New(sgin.Config{
		Mode:    gin.DebugMode,
		Locales: []language.Tag{language.Chinese, language.Korean},
		OpenAPI: oa.New(func(c *oa.Config) {}),
		Recovery: func(c *sgin.Ctx, out, s string) {
			fmt.Println(out)
		},
	})

	r.GET("/users/:id", func(c *sgin.Ctx, p User) Response {
		return Response{}
	})

	_ = r.Run(":8080")
}
