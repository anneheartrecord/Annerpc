package Annerpc

import (
	"Annerpc/codec"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// Call represents an active RPC
type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{} //arguments to the fuction
	Reply         interface{} //reply from the function
	Error         error       // if error occurs, it will be set
	Done          chan *Call  //send signal when call is complete
}

type Client struct {
	cc       codec.Codec //the message Codec,to marshal the sending request,and unmarshal the received response
	opt      *Option
	sending  sync.Mutex   //make sure send request one by one
	header   codec.Header //when sending request need a header
	mu       sync.Mutex
	seq      uint64           //unique seq
	pending  map[uint64]*Call //save the untreated requests
	closing  bool             //user has called Close
	shutdown bool             //server has told us to stop
}

func (call *Call) done() {
	call.Done <- call
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

// Close the connection

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// if the connection has been closed
	if c.closing {
		return ErrShutdown
	}
	c.closing = true
	return c.cc.Close()
}

// IsAvailable return true if the client does work
func (c *Client) IsAvailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.shutdown && !c.closing
}

// registerCall add the call to the client.pending and update the client seq
func (c *Client) registerCall(call *Call) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing || c.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = c.seq
	c.pending[call.Seq] = call
	c.seq++
	return call.Seq, nil
}

// removeCall according the seq remove the call from client.pending
func (c *Client) removeCall(seq uint64) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()
	call := c.pending[seq]
	delete(c.pending, seq)
	return call
}

// terminateCalls be used when server or client occurs error,
// set the shutdown be true and notice all calls in the client.pending
func (c *Client) terminateCalls(err error) {
	c.sending.Lock()
	defer c.sending.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdown = true
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
}

func (c *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		// if header wrong then break
		if err = c.cc.ReadHeader(&h); err != nil {
			break
		}
		// resolve the call ,so need remove it
		call := c.removeCall(h.Seq)
		switch {
		// it means that write partially failed
		//and call was already removed
		case call == nil:
			err = c.cc.ReadBody(nil)
			// the error is not empty means the server occurs error
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = c.cc.ReadBody(nil)
			call.done()
			// read the reply from the message body
		default:
			err = c.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body" + err.Error())
			}
			call.done()
		}
	}
	// error occurs,so terminateCalls pending calls
	c.terminateCalls(err)
}

func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client:codec error:", err)
		return nil, err
	}
	// send options with server
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client:options error:", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

// make a instance of Client
func newClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:     1, //seq starts with1,0 means invalid call
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}
