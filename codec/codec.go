package codec

import "io"

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

// Header is the Service Header struct
// contains ServiceMethod Seq and Error
// the Seq is Unique and the error field will be
// written on the server side if here is something wrong
type Header struct {
	ServiceMethod string //服务名和方法名
	Seq           uint64 //请求序号
	Error         string //错误信息  客户端置空 服务端写入
}

// Codec is an encoding and decoding interface
// ReadHeader read the decoded message header
// ReadBody read the decoded message body
// Write info to the header and the body
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// NewCodecFUnc is an alias of a function which
// args is io.ReadWriteCloser and return Codec

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

//NewCodecFuncMap map [string] func(io.ReadWriteCloser) Codec
var NewCodecFuncMap map[Type]NewCodecFunc

// init the Map
func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
	NewCodecFuncMap[JsonType] = NewGobCodec
}
