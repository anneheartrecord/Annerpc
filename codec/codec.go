package codec

import "io"

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

//  Header 表示服务头部的结构体
type Header struct {
	ServiceMethod string //服务名和方法名
	Seq           uint64 //请求序号
	Error         string //错误信息  客户端置空 服务端写入
}

// Codec 是对消息进行编解码的接口 分别有三个方法
// ReadHeader 读出解码之后的头
// ReadBody 读出解码之后的body
// Write 写入数据 header和body
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
