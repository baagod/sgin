package main

import (
	"fmt"

	"github.com/baagod/sgin"
	"github.com/gin-gonic/gin"
)

func main() {
	r := sgin.New(sgin.Config{
		Mode: gin.DebugMode,
	})

	r.GET("test", func(c *sgin.Ctx) {
		fmt.Println("test")
	})

	r.GET("test/prefix/c", func(c *sgin.Ctx) {
		fmt.Println("test/c")
	})

	g := r.Group("test/prefix", func(c *sgin.Ctx) {
		fmt.Println("test/prefix")
	})

	g.GET("a", func(c *sgin.Ctx) {
		fmt.Println("test/prefix/a")
	})

	g.GET("b", func(c *sgin.Ctx) {
		fmt.Println("test/prefix/b")
	})

	_ = r.Run()
}
