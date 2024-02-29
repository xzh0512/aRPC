package main

import (
	"fmt"
	"time"
)

func main() {
	nums := []int{1, 2, 3, 4, 5}
	//然而，当我们在闭包中引用外部的变量时，闭包会捕获该变量的引用，而不是复制其值。
	//这意味着闭包可以访问和修改外部变量的状态，而不仅仅是传递副本
	//所以当num改变的时候，前面闭包里的num也发生了变化，在执行时就不确定了
	for _, num := range nums {
		go func() {
			fmt.Println(num)
		}()
	}
	// 等待一段时间以使所有 goroutine 完成输出
	time.Sleep(time.Second)
}
