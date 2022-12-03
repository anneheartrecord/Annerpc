package Annerpc

import (
	"Annerpc/codec"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

//Client use Json encode the Option and the encode method of
//subsequent header and body is up to the Option
//Server first decode the option and use the CodeType to
//decode the subsequent header and body

type Option struct {
	MagicNumber int        //marks this is a rpc request
	CodecType   codec.Type //client can choose different Codec to encode body
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

var DefaultOption = &Option{ //default option use the Gob to encode and decode information
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server represents a rpc server
type Server struct{}

// NewServer  returns a new Server
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server
var DefaultServer = NewServer()

// Accept accepts connections on the listener
// and serves requests for each incoming connection
func (server *Server) Accept(lis net.Listener) {
	for {
		//wait the socket connecting and resolve
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server:accept error:", err)
			return
		}
		// use goroutine to resolve the connecting
		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

// ServeConn runs the server on a single connection
// ServeConn blocks,serving the connetcion until the client hangs up
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	//decode the option  and save it in the opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server:options error:", err)
		return
	}
	//check the magicnumber and CodecType is ok or not
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server:invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server:invalid codec type %s", opt.CodecType)
		return
	}
	// use serveCodec to resolve the next
	server.serveCodec(f(conn))
}

func (server *Server) serveCodec(cc codec.Codec) {
	//a mutex to make sue to send a complete response
	sending := new(sync.Mutex)
	//wait until all request are handled
	wg := new(sync.WaitGroup)
	for {
		//read the socket request
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			//send response must use the lock because the response need
			//be sent one by one
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		//concurrent processing handle the request
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	// ReadHeader until EOF
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server:read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server:read argv err:", err)
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	// use lock to make sure response one by one
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server:write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("rpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
