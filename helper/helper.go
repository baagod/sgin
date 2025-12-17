package helper

import (
    "reflect"
    "strings"
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
    return strings.ToUpper(s[:1]) + s[1:]
}
