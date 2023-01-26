package main

import (
	"context"
	"encoding/binary"
	"io"
	"log"

	"git.maharshi.ninja/root/rss2email/structures"
	"github.com/ugorji/go/codec"
	"nhooyr.io/websocket"
)

type MessageInfo struct {
	ID        uint32
	RequestID uint32
}

func (c *connection) readMessage(inf interface{}) (*MessageInfo, bool) {
	mi, buf, ok := c.readMessageInfo()
	if !ok {
		return nil, false
	}
	ok = c.decodeToInterface(buf, inf)
	if !ok {
		return nil, false
	}

	return mi, true
}

func (c *connection) readMessageInfo() (*MessageInfo, []byte, bool) {
	mtype, rdr, err := c.conn.Reader(context.TODO())
	if mtype != websocket.MessageBinary || err != nil {
		c.writeMessage(false, nil, structures.ErrorMessage{
			Code: structures.ErrorWhileDecoding,
		})
		return nil, nil, false
	}
	mi := new(MessageInfo)

	{
		buf := make([]byte, 8)
		_, err = io.ReadFull(rdr, buf)
		if err != nil {
			c.writeMessage(false, nil, structures.ErrorMessage{
				Code:    structures.ErrorWhileDecoding,
				Message: err.Error(),
			})
			return nil, nil, false
		}
		mi.ID = binary.LittleEndian.Uint32(buf[0:4])
		mi.RequestID = binary.LittleEndian.Uint32(buf[4:8])
	}

	rest, err := io.ReadAll(rdr)
	if err != nil {
		c.writeMessage(false, nil, structures.ErrorMessage{
			Code:    structures.ErrorWhileDecoding,
			Message: err.Error(),
		})
	}

	return mi, rest, true
}

func (c *connection) decodeToInterface(buf []byte, inf interface{}) bool {
	d := codec.NewDecoderBytes(buf, c.a.codecHandle)
	err := d.Decode(inf)
	if err != nil {
		log.Printf("Errored while decoding: %s\n", err.Error())
		c.writeMessage(false, nil, structures.ErrorMessage{
			Code:    structures.ErrorWhileDecoding,
			Message: err.Error(),
		})
		return false
	}
	return true
}

func (c *connection) writeError(m *MessageInfo, code structures.ErrorCode, err error) {
	c.writeMessage(false, m, structures.ErrorMessage{
		Code:    code,
		Message: err.Error(),
	})
}

func (c *connection) writeMessage(ok bool, m *MessageInfo, data interface{}) {
	wr, err := c.conn.Writer(context.TODO(), websocket.MessageBinary)
	defer func() {
		err := wr.Close()
		if err != nil {
			log.Printf("Error while writing: %s\n", err.Error())
			return
		}
	}()
	if err != nil {
		log.Printf("Error while writing: %s\n", err.Error())
		c.conn.Close(websocket.StatusInternalError, "???")
		return
	}

	{
		buf := make([]byte, 5)
		if m != nil {
			binary.LittleEndian.PutUint32(buf, m.ID)
		}
		if ok {
			buf[4] = 0xFF
		}
		_, err = wr.Write(buf)
		if err != nil {
			log.Printf("Error while writing: %s\n", err.Error())
			return
		}
	}

	enc := codec.NewEncoder(wr, c.a.codecHandle)
	err = enc.Encode(data)
	if err != nil {
		log.Printf("Error while writing: %s\n", err.Error())
		return
	}
}
