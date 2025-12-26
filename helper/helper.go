package helper

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

func DeRef(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func UpperFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// Convert 将输入值 v 转换为指定的反射类型 t。
// 它通过全递归设计，统一处理了指针解包、指针包装、切片重组和标量转换。
func Convert(t reflect.Type, field string, v any) any {
	if v == nil {
		return nil
	}

	vv := reflect.ValueOf(v)
	if vv.Type() == t {
		return v
	}

	// 如果底层类型兼容则直接转换，例如: json.RawMessage -> []byte。
	if vv.Type().ConvertibleTo(t) {
		return vv.Convert(t).Interface()
	}

	// 源数据解包: 处理源数据中的指针或接口。
	// 通过循环剥壳，直到见到底层的具体数值（例如将 *float64 剥开得到 float64）。
	for vv.Kind() == reflect.Ptr || vv.Kind() == reflect.Interface {
		if vv.IsNil() {
			return nil // 这里应该返回 nil
		}
		vv = vv.Elem() // 获取指针指向的值或接口存储的实际值
	}

	// 目标类型包装：如果目标字段类型是指针（例如 *int）。
	// 策略：先递归转换出底层类型的值（int），然后再为其创建并填充指针。
	// 这种设计可以自然支持多级指针（如 **int）。
	if t.Kind() == reflect.Ptr {
		elem := t.Elem()
		val := Convert(elem, field, vv.Interface())
		if val == nil {
			return nil
		}
		ptr := reflect.New(elem) // 反射创建指针: 相当于 ptr := new(T); *ptr = val
		ptr.Elem().Set(reflect.ValueOf(val))
		return ptr.Interface()
	}

	// 切片递归重组：处理切片（例如 []int 或 [][]int）。
	// 因为 Go 的切片类型不能直接强转（如 []any 不能转 []int），
	// 必须创建一个新容器，并将旧容器里的每个元素逐一递归转换后存入新容器。
	if t.Kind() == reflect.Slice && (vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array) {
		// 创建一个新的目标类型切片
		s := reflect.MakeSlice(t, 0, vv.Len())

		for i := 0; i < vv.Len(); i++ {
			// 关键：递归调用 convertType 转换元素。
			// 这种递归让它可以完美支持多维数组。
			s = reflect.Append(s, reflect.ValueOf(Convert(
				t.Elem(),
				fmt.Sprintf("%s[%d]", field, i),
				vv.Index(i).Interface(),
			)))
		}

		return s.Interface()
	}

	// 标量强转: 最后的底线，处理基本类型转换。
	// 此时 t 一定不是指针或切片，vv 也一定是剥开后的具体数值。
	// 常见场景：将 JSON 解析出的 float64(20.0) 转换为目标 int(20)。
	if !vv.Type().ConvertibleTo(t) {
		panic(fmt.Errorf("unable to convert value %v (%T) for field %q to %v", v, v, field, t))
	}

	return vv.Convert(t).Interface()
}

// CamelCase 使用大驼峰命名 s 并返回
func CamelCase(s string) string {
	// 遍历字符串 s，每当遇到返回 true 的字符，就把它当做分隔符切断。
	words := strings.FieldsFunc(s, func(r rune) bool {
		// 这里定义了分隔符：横线(-)、下划线(_)、点(.)
		return r == '-' || r == '_' || r == '.'
	})

	// 遍历切割出来的单词
	var sb strings.Builder
	for _, w := range words {
		if len(w) > 0 { // 首字母大写 + 剩余部分不变。
			sb.WriteString(UpperFirst(w))
		}
	}

	return sb.String()
}
