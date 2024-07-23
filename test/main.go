package main

import (
	"github.com/baagod/sgin"
	"github.com/gin-gonic/gin"
)

func main() {
	r := sgin.New(sgin.Config{
		Mode: gin.DebugMode,
	})

	r.Run()
}
