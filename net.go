package goubus

import (
	"io"
	//	"log"
	"net"
	"time"
)

func connect(path string) (*net.UnixConn, []byte, error) {
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, nil, err
	}

	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, nil, err
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * 60))

	bh := make([]byte, msgHeadSize)

	_, err = io.ReadFull(conn, bh)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, bh, nil
}

func discardRead(conn net.Conn, size uint32) error {
	b := make([]byte, size)
	_, err := io.ReadFull(conn, b)
	if err != nil {
		return err
	}

	return nil
}

func (ctx *UbusContext) sendMsg(b []byte) error {
	_, err := ctx.conn.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (ctx *UbusContext) recvMsg() (*msgHead, []byte, error) {
	b := make([]byte, msgHeadSize)
	_, err := io.ReadFull(ctx.conn, b)
	if err != nil {
		return nil, nil, err
	}
	var head msgHead

	head.unmarshal(b)

	padDataLen := roundUpLen(head.dataLen)

	//log.Printf("dataLen = %d, padDataLen = %d\n", head.dataLen, padDataLen)

	data := make([]byte, padDataLen)

	_, err = io.ReadFull(ctx.conn, data)
	if err != nil {
		return &head, nil, err
	}

	return &head, data, nil
}
