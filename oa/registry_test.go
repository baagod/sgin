package oa

import (
	"fmt"
	"reflect"
	"testing"
)

// AppConfig 包含了两个“值是匿名结构体”的 Map
type AppConfig struct {
	Settings map[string]struct {
		Timeout int `json:"timeout"`
	} `json:"settings"`

	Metadata map[string]struct {
		ID string `json:"id"`
	} `json:"metadata"`
}

func TestName(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n❌ 捕获到预期中的 Panic:\n%v\n", r)
		}
	}()

	// 初始化注册表
	registry := NewRegistry("/components/schemas/", DefaultSchemaNamer)

	fmt.Println("🚀 开始注册 AppConfig...")

	// 触发注册逻辑
	registry.Schema(reflect.TypeOf(AppConfig{}))

	for key := range registry.schemas {
		fmt.Println(key)
	}

	fmt.Println("✅ 注册成功 (如果没有看到这行，说明 Panic 了)")
}
