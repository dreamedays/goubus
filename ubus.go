package goubus

import (
	"fmt"
	//	"log"
	"net"
)

type UbusContext struct {
	localID uint32
	conn    *net.UnixConn
	seq     uint16
}

func Connect(path string) (*UbusContext, error) {
	conn, bh, err := connect(path)
	if err != nil {
		return nil, err
	}

	// parse head
	var head msgHead
	if err = head.unmarshal(bh); err != nil {
		return nil, err
	}

	if head.msgType != uint8(UBUS_MSG_HELLO) {
		return nil, fmt.Errorf("head msg type error")
	}

	//log.Printf("peer id %d\n", head.peerID)
	//log.Printf("data len %d\n", head.dataLen)

	if head.peerID == 0 {
		return nil, fmt.Errorf("head peer id is zero")
	}

	if head.dataLen > 0 {
		discardRead(conn, head.dataLen)
	}

	return &UbusContext{head.peerID, conn, 0}, nil
}

func (ctx *UbusContext) DisConnect() {
	if ctx.conn != nil {
		ctx.conn.Close()
		ctx.conn = nil
	}

	ctx.localID = 0
}

func (ctx *UbusContext) LookupID(object string) (uint32, error) {
	bb := NewBlobBuf()

	bb.putString(UBUS_ATTR_OBJPATH, object)

	ba, err := ctx.request(UBUS_MSG_LOOKUP, bb, 0)
	if err != nil {
		return 0, err
	}

	if ba[UBUS_ATTR_OBJID] == nil {
		return 0, fmt.Errorf("not contain objid in response")
	}

	id := ba[UBUS_ATTR_OBJID].getUint32()

	return id, nil
}

func (ctx *UbusContext) InvokeByID(peerID uint32, method string, bb *blobBuf) (*blobAttr, error) {

	bbh := NewBlobBuf()

	bbh.putUint32(UBUS_ATTR_OBJID, peerID)

	bbh.putString(UBUS_ATTR_METHOD, method)

	if bb == nil {
		// 需要构造一个空的blobBuf
		//log.Printf("make a empty blobBuf\n")
		bb = NewBlobBuf()
	}

	bbh.putData(bb)

	//log.Printf("data bb dataLen %d --->\n", bb.dataLen)
	//hexdump(bb.data[msgHeadSize:bb.dataLen])

	//log.Printf("send invoke request\n")
	ba, err := ctx.request(UBUS_MSG_INVOKE, bbh, peerID)
	if err != nil {
		return nil, err
	}

	//log.Printf("recv invoke response\n")
	_, resp, err := ctx.recvMsg()
	if err != nil {
		return nil, err
	}

	//log.Printf("parse invoke response message\n")
	ba, err = blobParse(resp)
	if err != nil {
		return nil, err
	}

	return ba[UBUS_ATTR_DATA], nil
}

func (ctx *UbusContext) InvokeByName(object, method string, bb *blobBuf) (*blobAttr, error) {

	id, err := ctx.LookupID(object)
	if err != nil {
		return nil, err
	}

	//log.Printf("object peerID 0x%x\n", id)

	return ctx.InvokeByID(id, method, bb)
}
