package main

import (
	"fmt"
	"reflect"
)

type MyStruct struct {
	Value int
}

func main() {
	s := MyStruct{Value: 42}

	// 尝试对非指针类型的值使用 reflect.Indirect()
	value := reflect.Indirect(reflect.ValueOf(s)).Type().Name() // 并不会导致错误

	fmt.Println(value)
}
