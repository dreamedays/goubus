package goubus

// 往ubusd发送请求
func (ctx *UbusContext) request(msgType UbsMsgType, bb *BlobBuf, peerID uint32) (*msgHead, []*BlobAttr, error) {
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
		return nil, nil, err
	}

	respHead, data, err := ctx.recvMsg()
	if err != nil {
		return nil, nil, err
	}

	//log.Printf("respHead, type %d\n", respHead.msgType)
	//log.Printf("response message head hexdump:\n")
	//hexdump(data)

	ba, err := blobBytesParse(data)
	if err != nil {
		return respHead, nil, err
	}

	return respHead, ba, nil
}
