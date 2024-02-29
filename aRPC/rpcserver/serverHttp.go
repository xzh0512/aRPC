package rpcserver

import (
	"io"
	"log"
	"net/http"
)

//1.客户端向 RPC 服务器发送 CONNECT 请求
//CONNECT 10.0.0.1:9999/_geerpc_ HTTP/1.0
//RPC 服务器返回 HTTP 200 状态码表示连接建立。
//HTTP/1.0 200 Connected to Gee RPC
//3.客户端使用创建好的连接 发送 RPC 报文，先发送 Option，再发送 N 个请求报文，服务端处理 RPC 请求并响应。

// 创建几个常量
const (
	connected        = "200 Connected to Gee RPC"
	defaultRpcPath   = "/_geerpc_"
	defaultDebugPath = "/debug/geerpc"
)

// 响应Rpc请求,实现了Handler接口。
// 需要实现的功能有：接收connect请求，回复请求，下发conn到业务层
func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//消息错误
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must CONNECT\n")
		return
	}
	//关闭conn不会关闭掉现在的http连接
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	_, _ = io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")
	server.Parser(conn)
}

// HandleHttp 向httpserver注册上面的服务，上面的服务实现了http的handle接口
func (server *Server) HandleHttp() {
	http.Handle(defaultRpcPath, server)
	http.Handle(defaultDebugPath, debugHTTP{server})
	log.Println("rpc server debug path:", defaultDebugPath)
}

// HandleHttp 启动默认的服务器
func HandleHttp() {
	DefaultServer.HandleHttp()
}
