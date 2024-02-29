package main

import (
	"context"
	"time"
)

type Demo struct {
	ch chan int
	i  int
}

func Go(a, b int, ch chan int) *Demo {
	demo := &Demo{
		i:  a + b,
		ch: ch,
	}

	time.Sleep(10 * time.Second)
	demo.ch <- demo.i

	return demo
}
func main() {
	ch := make(chan int, 1)
	ctx, _ := context.WithTimeout(context.Background(), time.Second*6) //但是使用上下文就不会这样，从创建到结束会直接提示
	ret := Go(2, 3, ch)                                                // 会一直阻塞在这里等待协程返回
	a := 3
	println(a)
	select {
	case <-ret.ch:
		println("yes")
	case <-ctx.Done():
		println("error")

	case <-time.After(5 * time.Second):
		println("aftertime")

	}
}
