package main

import (
	"github.com/baagod/sgin"
	"github.com/baagod/sgin/oa"
)

type User struct {
	Name string `json:"name"`
}

func main() {
	r := sgin.New(sgin.Config{
		OpenAPI: oa.New(),
	})

	r.GET("/test", func(c *sgin.Ctx, u User) User {
		return u
	})

	r.Run(":8080")
}
