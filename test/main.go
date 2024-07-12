package main

import (
	"errors"

	"github.com/baagod/sgin"
	"github.com/gin-gonic/gin"
)

func main() {
	r := sgin.New(sgin.Config{
		Mode: gin.ReleaseMode,
	})

	r.GET("test", func(c *sgin.Ctx) error {
		return errors.New("error")
	})

	r.Run(":9852")
}
