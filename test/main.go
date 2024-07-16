package main

import (
	"fmt"

	"github.com/baagod/sgin"
)

func main() {
	var r sgin.Response
	r2 := r.SetStatus(1)

	fmt.Printf("r, r2: %T, %T\n", r, r2)

	// r := sgin.New(sgin.Config{
	// 	Mode: gin.ReleaseMode,
	// })
	//
	// r.GET("test", func(c *sgin.Ctx) error {
	// 	return errors.New("error")
	// })
	//
	// r.Run(":9852")
}
