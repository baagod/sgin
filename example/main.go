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

func level2() {
	panic("假设这里是 1/0 或其他 panic")
}

func level1() {
	fmt.Println("这里是 level1")
	level2()
}

func main() {
	r := sgin.New(sgin.Config{
		Mode:    gin.DebugMode,
		Locales: []language.Tag{language.Chinese, language.Korean},
		OpenAPI: oa.New(oa.Config{}),
		Recovery: func(c *sgin.Ctx, out, s string) {
			fmt.Println(out)
		},
	})

	r.GET("/users/:id", func(c *sgin.Ctx, p User) Response {
		level1()
		return Response{}
	})

	_ = r.Run(":8080")
}
