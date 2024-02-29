package rpcserver

import (
	endecode "aRPC/edcode"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const MagicData = 0x3bef5c

// Option 定义报文头
// 协商编解码方式,这个是最先发过去的。
// 自定义通信方式，而http就是规定死的方式，前面不需要 Option 规定哪种方式了
// GeeRPC 客户端固定采用 JSON 编码 Option，
// 后续的 header 和 body 的编码方式由 Option 中的 CodeType 指定
type Option struct {
	MagicInt       int
	CodeType       endecode.Type
	ConnectTimeout time.Duration // 0 means no limit
	HandleTimeout  time.Duration
}

// DefaultOption 设置一个默认格式
var DefaultOption = &Option{
	MagicInt:       MagicData,
	CodeType:       endecode.GobType,
	ConnectTimeout: time.Second * 10, //设置10秒钟的连接超时
}

// Server 服务器
type Server struct {
	serviceMap sync.Map
}

func (server *Server) Register(instance interface{}) error {
	s := newService(instance)
	if _, loaded := server.serviceMap.LoadOrStore(s.name, s); loaded {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

// Register publishes the receiver's methods in the DefaultServer.
func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }

var DefaultServer = NewServer()

func NewServer() *Server {
	return &Server{}
}

// 根据头部的方法名找到对应的服务，以及方法
func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	//serviceMethod: service.method 可以用.划分成两部分
	parts := strings.Split(serviceMethod, ".")
	if len(parts) != 2 {
		return nil, nil, errors.New("false serviceMethod " + serviceMethod)
	}
	serverName := parts[0]
	value, ok := server.serviceMap.Load(serverName)
	if ok == false {
		return nil, nil, errors.New(serverName + " is not exist")
	}
	svc = value.(*service)
	methodName := parts[1]
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return //语法糖
}

func ListenAndHandle(s *Server, listener net.Listener) {
	s.Accept(listener)
}
func (server *Server) Accept(lis net.Listener) {
	for {
		//对于每一个客户端给一个连接
		conn, err := lis.Accept()
		if err != nil {
			log.Println("connect error")
			return
		}
		go server.Parser(conn)
	}
}

func (server *Server) Parser(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	/*54-68行，只解析一次(option)：
	| Option | Header1 | Body1 | Header2 | Body2 | ...*/
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("json parser err")
		return
	}
	if opt.MagicInt != MagicData {
		log.Printf("rpc server: invalid magic number %x\n", opt.MagicInt)
	}
	//找到编解码注册表的函数
	f := endecode.NewCodecFuncMap[opt.CodeType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %server", opt.CodeType)
		return
	}
	//具体报文解析：header+body
	//f(conn)->Codec创造一个消息解译码器
	server.ServeCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) ServeCodec(c endecode.Codec) {
	sending := new(sync.Mutex) // make sure to send a complete response
	wg := new(sync.WaitGroup)  // wait until all request are handled
	for {
		//解析消息，最后没消息可读时会自动关闭连接
		reply, err := server.ParserReply(c)
		if err != nil {
			if reply == nil {
				//只有关闭了连接就会出现读到文件尾的错误。
				if err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
					log.Println("rpc server: read header error:", err)
				}
				log.Println("handle done")
				break
			}
			reply.h.Error = err.Error()
			sending.Lock()
			err := c.WriteHeaderAndBody(reply.h, invalidRequest)
			if err != nil {
				log.Println("rpc server: write response error:", err)
			}
			sending.Unlock()
			continue
		}
		//处理消息
		wg.Add(1)
		go func() {
			defer func() {
				recover()
				wg.Done()
			}()
			server.Handle(c, reply, sending, 1)
		}()
	}
	wg.Wait()
	_ = c.Close()
}

type Reply struct {
	h         *endecode.Header // header of request
	argv, msg reflect.Value    // reflect.Value用于表示一个值的反射信息
	mtype     *methodType
	svc       *service
}

func (server *Server) ParserReply(c endecode.Codec) (*Reply, error) {
	var h = &endecode.Header{}
	if err := c.ReadHeader(h); err != nil {
		//把余下的读掉，不然再读head会类型不匹配,
		//发是完整的发过来读也应该如此
		_ = c.ReadBody(nil)
		return nil, err
	}
	req := &Reply{h: h}
	var err error
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		//把余下的读掉，不然再读head会类型不匹配,
		//发是完整的发过来读也应该如此
		_ = c.ReadBody(nil)
		return req, err
	}
	//创建两个空参数
	req.argv = req.mtype.newArgv()
	req.msg = req.mtype.newReply()

	// make sure that argvi is a pointer, ReadBody need a pointer as parameter
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	//读入参数信息
	if err := c.ReadBody(argvi); err != nil {
		log.Println("rpc server: read argv err:", err)
		return nil, err
	}
	return req, nil
}
func (server *Server) sendRequest(c endecode.Codec, h *endecode.Header, body interface{}, sending *sync.Mutex) {
	defer sending.Unlock()
	sending.Lock()
	if err := c.WriteHeaderAndBody(h, body); err != nil {
		log.Println("rpc server sendRequest error:", err)
	}

}
func (server *Server) Handle(c endecode.Codec, reply *Reply, sending *sync.Mutex, timeout time.Duration) {
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := reply.svc.call(reply.mtype, reply.argv, reply.msg)
		called <- struct{}{}
		if err != nil {
			reply.h.Error = err.Error()
			server.sendRequest(c, reply.h, invalidRequest, sending)
			log.Println("rpc server call error:", err)
			sent <- struct{}{}
			return
		}
		server.sendRequest(c, reply.h, reply.msg.Interface(), sending)
		sent <- struct{}{}
	}()
	if timeout == 0 {
		<-called
		<-sent
		return
	}
	select {
	case <-time.After(timeout):
		reply.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.sendRequest(c, reply.h, invalidRequest, sending)
	case <-called:
		<-sent
	}
}
