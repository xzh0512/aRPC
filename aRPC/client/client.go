package client

import (
	"aRPC/edcode"
	"aRPC/rpcserver"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Call func (t *T) MethodName(argType T1, replyType *T2) error
type Call struct {
	ServerMethod string
	Seq          uint64
	Args         interface{}
	Reply        interface{}
	Error        error
	Done         chan *Call //是否已经完成
}

// 将完成的call塞入管道
func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	c       edcode.Codec
	opt     *rpcserver.Option
	sending sync.Mutex //收发过程的锁
	header  edcode.Header
	mu      sync.Mutex //客户端内部锁，用来锁住客户端成员修改的操作

	seq     uint64
	pending map[uint64]*Call //存储未处理完的请求，键是编号:Seq，值是 Call 实例。

	closing  bool //正常关闭
	shutdown bool //异常关闭
}

var _ io.Closer = (*Client)(nil)

// Close the connection
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.c.Close()
}

// IsAvailable return true if the client does work
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

var ErrShutdown = errors.New("connection is shut down")

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++ //注册一个分配一个seq，从0开始分配
	return call.Seq, nil
}
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

// 在客户端一启动就会持续监听服务端发过来的请求，发生错误就会终止并报告错误
// 只有一个读口，不需要加锁
func (client *Client) receive() {
	var err error
	for err == nil {
		var h edcode.Header
		//主要阻塞在读报头这里
		if err = client.c.ReadHeader(&h); err != nil {
			//是否有这个消息，得看主进程有没有提前退出
			fmt.Println("Err:", err)
			break
		}
		//client 里面的序列号是用来分配的
		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			err = client.c.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			_ = client.c.ReadBody(nil)
			call.done()
		default:
			err = client.c.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	// error occurs, so terminateCalls pending calls
	client.terminateCalls(err)
}

// NewClient 在新建客户端的时候就启动了receive
func NewClient(conn net.Conn, option *rpcserver.Option) (*Client, error) {
	f := edcode.NewCodecFuncMap[option.CodeType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", option.CodeType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}
	if err := json.NewEncoder(conn).Encode(option); err != nil {
		log.Println("rpc client: options error: ", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), option), nil
}
func newClientCodec(codec edcode.Codec, option *rpcserver.Option) *Client {
	client := &Client{
		seq:     1, // seq starts with 1, 0 means invalid call
		c:       codec,
		opt:     option,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

// 可变参数，1或者0
func parseOptions(opts ...*rpcserver.Option) (*rpcserver.Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return rpcserver.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicInt = rpcserver.MagicData
	if opt.CodeType == "" {
		opt.CodeType = rpcserver.DefaultOption.CodeType
	}
	return opt, nil
}

type clientResult struct {
	client *Client
	err    error
}
type NewClientFunc func(conn net.Conn, option *rpcserver.Option) (*Client, error)

func dialTimeout(f NewClientFunc, network, addr string, opts ...*rpcserver.Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout(network, addr, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	ch := make(chan clientResult)
	go func() {
		c, err2 := f(conn, opt)
		ch <- clientResult{client: c, err: err2}
	}()
	if opt.ConnectTimeout == 0 {
		ret := <-ch
		return ret.client, ret.err
	}
	//超时选项,客户端创建超时
	select {
	case ret := <-ch:
		return ret.client, ret.err
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	}
}
func Dial(network, address string, opts ...*rpcserver.Option) (*Client, error) {
	return dialTimeout(NewClient, network, address, opts...)
}
func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		return
	}

	//发送头
	client.header.Seq = seq
	client.header.ServiceMethod = call.ServerMethod
	//消息体：call.Args
	if err := client.c.WriteHeaderAndBody(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Go 需要新建一个call，并发送出去
func (client *Client) Go(MethodName string, args, reply interface{}, ch chan *Call) *Call {
	if ch == nil {
		ch = make(chan *Call, 10)
	} else if cap(ch) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServerMethod: MethodName,
		Args:         args,
		Reply:        reply,
		Done:         ch,
	}
	//client.Go() 函数里的 client.send() ，
	//是否应该为 go client.send() ？
	//我认为返回 call 不需要等待 client.send() 执行完。
	client.send(call)
	return call
}

// Call 参数和响应都是空接口类型
// 异步请求确实有很多种其他的方式，
// 但是 client.Go 的好处在于参数 ch chan *Call 可以自定义缓冲区的大小，
// 可以给多个 client.Go 传入同一个 chan 对象，从而控制异步请求并发的数量。
func (client *Client) Call(ctx context.Context, MethodName string, args, reply interface{}) error {
	call := client.Go(MethodName, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		log.Println("timeout")
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call1 := <-call.Done:
		return call1.Error
	}
}
