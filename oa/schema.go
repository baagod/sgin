package oa

import (
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/baagod/sgin/helper"
)

// JSON Schema 类型常量
const (
	TypeBoolean = "boolean"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeString  = "string"
	TypeArray   = "array"
	TypeObject  = "object"
)

// 特殊的 JSON Schema 格式
var (
	timeType       = reflect.TypeOf(time.Time{})
	ipType         = reflect.TypeOf(net.IP{})
	ipAddrType     = reflect.TypeOf(netip.Addr{})
	urlType        = reflect.TypeOf(url.URL{})
	rawMessageType = reflect.TypeOf(json.RawMessage{})
)

type Schema struct {
	Type                 any                `yaml:"type,omitempty"`
	Nullable             bool               `yaml:"-"`
	Title                string             `yaml:"title,omitempty"`
	Description          string             `yaml:"description,omitempty"`
	Ref                  string             `yaml:"$ref,omitempty"`
	Format               string             `yaml:"format,omitempty"`
	ContentEncoding      string             `yaml:"contentEncoding,omitempty"`
	Default              any                `yaml:"default,omitempty"`
	Items                *Schema            `yaml:"items,omitempty"`                // For arrays
	AdditionalProperties any                `yaml:"additionalProperties,omitempty"` // Schema or bool
	Properties           map[string]*Schema `yaml:"properties,omitempty"`
	Enum                 []any              `yaml:"enum,omitempty"`
	Required             []string           `yaml:"required,omitempty"`
}

func (s *Schema) MarshalYAML() (any, error) {
	if s.Nullable {
		s.Type = []any{s.Type, "null"}
	}
	return s, nil
}

// fieldInfo 用于存储字段的详细信息，包括其直接父级类型。
// 这在处理复杂的内嵌结构体时非常有用。
type fieldInfo struct {
	Parent reflect.Type
	Field  reflect.StructField
}

// getFields 通过广度优先搜索（BFS）遍历一个类型的所有字段，并在发现每个字段时调用回调函数。
// 它处理内嵌结构体，并通过 visited 集合避免无限递归。
// 使用迭代式 BFS 配合 head 索引，实现高效且清晰的队列操作。
func getFields(t reflect.Type, callback func(info fieldInfo)) {
	// 使用切片模拟队列，并用 head 索引追踪队列头部
	queue := []reflect.Type{t}
	// visited 集合用于防止对同一结构体类型的重复处理
	visited := map[reflect.Type]struct{}{t: {}}

	// 队列处理循环：head 索引在每次迭代中递增，len(queue) 会动态更新
	for head := 0; head < len(queue); head++ {
		currentTyp := queue[head] // 获取当前待处理的类型

		// 遍历当前类型的所有字段
		for i := 0; i < currentTyp.NumField(); i++ {
			f := currentTyp.Field(i)

			// 忽略非导出字段（小写字母开头），因为它们不会被 JSON 序列化
			if !f.IsExported() {
				continue
			}

			// 如果是内嵌字段（匿名字段），则需要进一步处理其内部结构
			if f.Anonymous {
				// 解引用以获取实际类型，因为内嵌字段可能是指针
				embeddedTyp := helper.DeRef(f.Type)

				// 只有当内嵌的是结构体且该类型尚未被访问过时，才将其加入队列等待处理
				if embeddedTyp.Kind() == reflect.Struct {
					if _, ok := visited[embeddedTyp]; !ok {
						visited[embeddedTyp] = struct{}{}
						queue = append(queue, embeddedTyp) // 将新类型加入队列尾部
					}
				}
				continue // 内嵌字段本身不直接作为 Schema 属性，而是其内部字段会通过回调处理
			}

			// 对于非内嵌的普通字段，执行传入的回调函数
			callback(fieldInfo{Parent: currentTyp, Field: f})
		}
	}
}

// parseTagValue 根据字段的 Schema 类型，将从 tag 读取的字符串值解析为正确的 Go 类型。
// 例如，对于一个 integer 字段，它会将 "123" 解析为数字 123。
func parseTagValue(value, fieldname string, s *Schema) any {
	// 1. 如果基础类型是 string，直接返回原始值，无需解析。
	if s.Type == TypeString {
		return value
	}

	// 特殊情况：字符串数组，带有逗号分隔且无引号。
	if s.Type == TypeArray && s.Items != nil && s.Items.Type == TypeString && value[0] != '[' {
		values := make([]string, 0)
		for _, v := range strings.Split(value, ",") {
			values = append(values, strings.TrimSpace(v))
		}
		return values
	}

	// 2. 对于所有其他类型，尝试使用 JSON 解码器进行解析
	var result any
	value = strings.TrimSpace(value)
	err := json.Unmarshal([]byte(value), &result)
	if err != nil {
		// 返回错误，以便调用者可以决定如何处理（例如，忽略无效的 tag 值）
		panic(fmt.Errorf("invalid %s tag value '%s' for field '%s': %w", s.Type, value, fieldname, err))
	}

	return result
}
