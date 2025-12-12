package helper

import "reflect"

func DeRef(t reflect.Type) reflect.Type {
    for t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    return t
}
