package main

import (
	"bytes"
	"fmt"
	"github.com/bytedance/sonic"
)

func main() {
	// r := sgin.New()
	//
	// r.GET("test", func(c *sgin.Ctx) {
	// 	fmt.Println(c.Args())
	// })
	//
	// r.Run(":9852")

	m := map[string]any{}
	dec := sonic.ConfigDefault.NewDecoder(bytes.NewReader(nil))
	dec.UseNumber()
	_ = dec.Decode(&m)
	// c.args = args

	fmt.Println(m, m == nil)
}
