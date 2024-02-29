package main

import (
	client2 "aRPC/client"
	"aRPC/rpcserver"
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type Bar int

func (b Bar) Timeout(argv int, reply *int) error {
	time.Sleep(time.Second * 5)
	*reply = 1
	return nil
}

func startServer(addr chan string) {
	var foo Foo
	if err := rpcserver.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	// pick a free port
	//创建并监听一个tcp连接，实际上面的传输协议可以自定义：http,rpc等
	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	rpcserver.HandleHttp()
	addr <- l.Addr().String()
	err = http.Serve(l, nil)
	if err != nil {
		log.Println("connect closed")
	}
}

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}
func call(addrCh chan string) {
	client, _ := client2.DailHttp("tcp", <-addrCh) //持续监听是在这里发生的
	//是因为发了关闭conn，服务端那边才会发 handle done.客户端再监听消息就是使用了已经关闭的请求
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call(context.Background(), "Fo.Sum", args, &reply); err != nil {
				log.Println("call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}
func main() {
	addr := make(chan string)
	go call(addr)
	startServer(addr)
}
