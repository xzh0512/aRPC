package main

import (
	"fmt"
	"reflect"
)

type Args struct {
	A, B int
}

type Foo struct{}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.A + args.B
	return nil
}

func main() {
	// 创建 Foo 类型的实例
	foo := Foo{}

	// 构造方法的参数
	args := Args{A: 2, B: 3}
	reply := 0

	// 获取 Sum 方法的反射值
	name, _ := reflect.TypeOf(foo).MethodByName("Sum")
	f := name.Func
	a := name.Type
	fmt.Println(a.NumIn(), a.NumOut())
	method := reflect.ValueOf(foo).MethodByName("Sum")
	fmt.Println(method.Type().NumIn(), method.Type().NumOut())
	// 构造输入参数的反射值切片
	in := []reflect.Value{
		reflect.ValueOf(foo),
		reflect.ValueOf(args),
		reflect.ValueOf(&reply),
	}

	// 调用方法
	result := f.Call(in)

	// 处理返回值
	if len(result) > 0 {
		err := result[0].Interface()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	fmt.Println("Reply:", reply) // 输出: Reply: 5
}
