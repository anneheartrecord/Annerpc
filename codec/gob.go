package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCodec 由一个通道一个缓冲一个编码器一个解码器组成
type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

//检查结构体是否实现接口
var _ Codec = (*GobCodec)(nil)

// NewGobCodec 创建一个Gob类型的编解码器
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

//实现方法

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h) //返回解码之后的头部
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body) //返回解码之后的body
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush() //写入数据
		if err != nil {
			_ = c.Close()
		}
	}()
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec:gob error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec:gob error encoding body:", err)
		return err
	}
	return nil
}

func (c *GobCodec) Close() error {
	return c.conn.Close() //关闭通道
}
