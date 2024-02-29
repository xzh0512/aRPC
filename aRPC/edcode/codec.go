package edcode

import "io"

// Header 表明了消息体里的信息，和自身信息
type Header struct {
	ServiceMethod string // format "Service.Method",请求方法
	Seq           uint64 // sequence number chosen by client
	Error         string
}
type Codec interface {
	io.Closer
	/*读取*/
	ReadHeader(header *Header) error
	ReadBody(interface{}) error //你并不知道body的具体实现
	/*写入*/
	WriteHeaderAndBody(header *Header, body interface{}) error
}

/*给Codec实现一个构造函数注册表*/
type NewCodecFunc func(conn io.ReadWriteCloser) Codec

type Type string

const GobType Type = "application/gob"

var NewCodecFuncMap map[Type]NewCodecFunc

// 初始化的注册方法
func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGob
}
