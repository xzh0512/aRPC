package client

import (
	"aRPC/rpcserver"
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const (
	connected        = "200 Connected to Gee RPC"
	defaultRpcPath   = "/_geerpc_"
	defaultDebugPath = "/debug/geerpc"
)

//客户端发送CONNECT连接，然后等待服务器回复
//下发conn到业务层

func NewHttpClient(conn net.Conn, opt *rpcserver.Option) (*Client, error) {
	//发送连接请求
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.1\n\n", defaultRpcPath))
	//使用 bufio.Reader 的主要目的是为了提高读取的性能和效率，
	//通过内部的缓冲区减少与底层连接的直接交互次数。
	//如果你已经确定不需要这种缓冲机制，直接使用 conn 作为参数也是可行的。
	response, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && response.Status == connected {
		return NewClient(conn, opt)
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + response.Status)
	}
	return nil, err
}

func DailHttp(network, address string, opts ...*rpcserver.Option) (*Client, error) {
	return dialTimeout(NewHttpClient, network, address, opts...)
}
func XDial(rpcAddr string, opts ...*rpcserver.Option) (*Client, error) {
	parts := strings.Split(rpcAddr, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("rpc client err: wrong format '%s', expect protocol@addr", rpcAddr)
	}
	protocol, addr := parts[0], parts[1]
	switch protocol {
	case "http":
		return DailHttp("tcp", addr, opts...)
	default:
		return Dial("tcp", addr, opts...)
	}
}
