package main

import (
	"fmt"
	"reflect"
)

func main() {
	str := "Hello, World!"
	a := 1
	//b := a.(interface{}) 类型断言,只适合接口类型
	b := interface{}(a) //强制类型转换，type(exp),类型(原实例)与C语言正好相反(type)exp
	v := reflect.ValueOf(b)
	fmt.Println(v.Type())
	// 将 string 类型的值转换为 interface{} 类型
	var i interface{} = str

	// 对 interface{} 类型的值进行反射
	value := reflect.ValueOf(i)

	fmt.Println("Type:", value.Type())    // 输出: string
	fmt.Println("Kind:", value.Kind())    // 输出: string
	fmt.Println("Value:", value.String()) // 输出: Hello, World!
}
