package main

import (
	"mime/multipart"

	"github.com/baagod/sgin"
)

type User struct {
	Name string `json:"name"`
}

type HelloReq struct {
	Name string `form:"name"`
}

type FileModel struct {
	File   *multipart.FileHeader `form:"file"`
	Age    int                   `form:"age"`
	Weight int64                 `form:"weight"`
	Child  int64                 `form:"child"` // 测试是否显示 format
}

func main() {
	r := sgin.New(sgin.Config{
		OpenAPI: sgin.NewAPI(),
	})

	r.POST("/hello", sgin.Ho(func(c *sgin.Ctx, req FileModel) error {
		return nil
	}))

	// r.GET("/doc", sgin.Ho(func(c *sgin.Ctx, _ struct{}) string {
	// 	return "ok"
	// }), func(op *sgin.Operation) {
	// 	op.Summary = "Doc Test"
	// })

	_ = r.Run(":8080")
}
