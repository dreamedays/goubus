package goubus

// 往ubusd发送请求
func (ctx *UbusContext) request(msgType ubsMsgType, bb *blobBuf, peerID uint32) ([]*blobAttr, error) {
	// 构造 msgHead
	var head msgHead
	head.version = 0
	head.msgType = byte(msgType)
	head.seq = ctx.seq
	ctx.seq++
	head.peerID = peerID
	head.dataLen = bb.dataLen

	head.marshal(bb.data)

	//log.Printf("request message hexdump:\n")
	//hexdump(bb.data[0:bb.dataLen])

	err := ctx.sendMsg(bb.data[0:bb.dataLen])
	if err != nil {
		return nil, err
	}

	_, data, err := ctx.recvMsg()
	if err != nil {
		return nil, err
	}

	//log.Printf("response message hexdump:\n")
	//hexdump(data)

	ba, err := blobParse(data)
	if err != nil {
		return nil, err
	}

	return ba, nil
}
