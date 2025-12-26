package main

import (
	"github.com/baagod/sgin"
)

type User struct {
	Name string `json:"name"`
}

type HelloReq struct {
	Name string `form:"name"`
}

func main() {
	r := sgin.New(sgin.Config{
		OpenAPI: sgin.NewAPI(),
	})

	r.GET("/hello", sgin.Ho(func(c *sgin.Ctx, req HelloReq) string {
		return "Hello " + req.Name
	}))

	r.GET("/doc", sgin.Ho(func(c *sgin.Ctx, _ struct{}) string {
		return "ok"
	})).Operation(func(op *sgin.Operation) {
		op.Summary = "Doc Test"
	})

	_ = r.Run(":8080")
}
