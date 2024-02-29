package main

import (
	"fmt"
	"reflect"
)

type m struct {
	s *string
}

func main() {
	s1 := "hello"
	a := &m{
		s: &s1,
	}
	fmt.Println(a)
	i := 1
	fmt.Println(reflect.New(reflect.ValueOf(i).Type()).Kind())
	//Elem()作用于指针类型
	//在 Go 的反射库中，Elem() 方法用于获取指针类型的元素类型。
	//它适用于 reflect.Type 和 reflect.Value 类型的指针。
	fmt.Println(reflect.TypeOf(&i).Elem())  //print int
	fmt.Println(reflect.ValueOf(&i).Elem()) //print 1
}
