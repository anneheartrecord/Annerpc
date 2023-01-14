package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCodec contains a conn and a buf used in communication
// and an encoder and decoder
type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

//check the GobCodec struct whether satisfied the Codec interface
var _ Codec = (*GobCodec)(nil)

// NewGobCodec accept the io.ReadWriteCloser args and return a Codec interface
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	// the connection has three  fields Reader Writer and Closer
	// create a buf
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

//realize the three Methods of Codec and use the GobCodec be the receiver

func (c *GobCodec) ReadHeader(h *Header) error {
	//return decoded header
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	//return decoded body
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		// write the buf data
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	// encode the header
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec:gob error encoding header:", err)
		return err
	}
	// encode the body
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec:gob error encoding body:", err)
		return err
	}
	return nil
}

func (c *GobCodec) Close() error {
	// close the connection
	return c.conn.Close()
}
