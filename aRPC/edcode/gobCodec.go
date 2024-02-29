package edcode

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCodec 设置消息编解码的，每当连接建立
// 会将连接写入，并初始化编解码管道
// 这个类实现了编解码的接口
type GobCodec struct {
	//连接
	conn io.ReadWriteCloser
	//写缓冲，提升性能
	buf    *bufio.Writer
	decode *gob.Decoder
	encode *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGob(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn) //改造一下连接
	return &GobCodec{
		conn: conn,
		buf:  buf,
		//同时进行读取和写入操作,因此需要两个对象
		decode: gob.NewDecoder(conn),
		encode: gob.NewEncoder(buf),
	}
}
func (g *GobCodec) Close() error {
	return g.conn.Close()
}

// read方法就只要传个空指针，从编码器的缓冲区取出数据
func (g *GobCodec) ReadHeader(header *Header) error {
	return g.decode.Decode(header)
}
func (g *GobCodec) ReadBody(body interface{}) error {
	return g.decode.Decode(body)
}

// write方法是将参数传入编码器的缓冲区进行译码
func (g *GobCodec) WriteHeaderAndBody(header *Header, body interface{}) (err error) {
	defer func() {
		_ = g.buf.Flush()
		if err != nil {
			_ = g.Close()
		}
	}()
	if err = g.encode.Encode(header); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err = g.encode.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}
