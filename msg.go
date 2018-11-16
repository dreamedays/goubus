package goubus

import (
	"encoding/binary"
	"fmt"
	"log"
	"unicode"
)

type ubsMsgType int

const (
	UBUS_MSG_HELLO ubsMsgType = iota
	UBUS_MSG_STATUS
	UBUS_MSG_DATA
	UBUS_MSG_PING
	UBUS_MSG_LOOKUP
	UBUS_MSG_INVOKE
	UBUS_MSG_ADD_OBJECT
	UBUS_MSG_REMOVE_OBJECT
	UBUS_MSG_SUBSCRIBE
	UBUS_MSG_UNSUBSCRIBE
	UBUS_MSG_NOTIFY
	UBUS_MSG_MONITOR

	UBUS_MSG_LAST
)

type ubusMsgAttr int

const (
	UBUS_ATTR_UNSPEC ubusMsgAttr = iota
	UBUS_ATTR_STATUS
	UBUS_ATTR_OBJPATH
	UBUS_ATTR_OBJID
	UBUS_ATTR_METHOD
	UBUS_ATTR_OBJTYPE
	UBUS_ATTR_SIGNATURE
	UBUS_ATTR_DATA
	UBUS_ATTR_TARGET
	UBUS_ATTR_ACTIVE
	UBUS_ATTR_NO_REPLY
	UBUS_ATTR_SUBSCRIBERS
	UBUS_ATTR_USER
	UBUS_ATTR_GROUP

	UBUS_ATTR_MAX
)

const msgHeadSize = 12

type msgHead struct {
	version uint8
	msgType uint8
	seq     uint16
	peerID  uint32
	dataLen uint32 // 数据长度，不包括本字段的长度
}

func (h *msgHead) unmarshal(b []byte) error {
	h.version = uint8(b[0])
	if h.version != 0 {
		return fmt.Errorf("msg head error")
	}

	h.msgType = uint8(b[1])

	h.seq = binary.BigEndian.Uint16(b[2:])

	h.peerID = binary.BigEndian.Uint32(b[4:])

	h.dataLen = binary.BigEndian.Uint32(b[8:])

	h.dataLen = (h.dataLen & 0x00ffffff) - 4

	return nil
}

func (h *msgHead) marshal(data []byte) {
	data[0] = h.version
	data[1] = h.msgType
	binary.BigEndian.PutUint16(data[2:], h.seq)
	binary.BigEndian.PutUint32(data[4:], h.peerID)
	binary.BigEndian.PutUint32(data[8:], h.dataLen-msgHeadSize+4)
}

func (h *msgHead) String() string {
	return fmt.Sprintf("ver: %d, type: %d, seq: %d, peerID: %d, dataLen %d", h.version, h.msgType, h.seq, h.peerID, h.dataLen)
}

func hexdump(b []byte) {
	var i, j int

	fmt.Printf("bb data len %d\n", len(b))

	fmt.Printf("        ")
	for j = 0; j < 16; j++ {
		fmt.Printf("%02d ", j)
	}
	fmt.Println()
	fmt.Printf("        ")
	for j = 0; j < 16; j++ {
		fmt.Printf("-- ")
	}
	fmt.Println()

	for i = 0; i < len(b); {
		fmt.Printf("0x%04x  ", i&0xf)
		for j = 0; i < len(b) && j < 16; {
			fmt.Printf("%02x ", b[i])
			i++
			j++
		}
		if j < 16 {
			pad := 16 - j
			for ; pad > 0; pad-- {
				fmt.Printf("   ")
			}
		}
		fmt.Printf("   ")
		i = i - j
		j = 0
		for i < len(b) && j < 16 {
			if unicode.IsPrint(rune(b[i])) {
				fmt.Printf("%c", b[i])
			} else {
				fmt.Printf(".")
			}
			i++
			j++
		}
		fmt.Println()
	}
}

type blobBuf struct {
	dataLen uint32
	data    []byte
}

func NewBlobBuf() *blobBuf {
	var bb blobBuf
	bb.data = make([]byte, 256)
	bb.dataLen = msgHeadSize

	return &bb
}

const blobRoudUpSize uint32 = 4

func roundUpLen(l uint32) uint32 {
	return (l + blobRoudUpSize - 1) & ^(blobRoudUpSize - 1)

}

func (bb *blobBuf) checkSize(a uint32) {
	if uint32(cap(bb.data)) < bb.dataLen+a {
		// 扩容
		fmt.Printf("enlarge the blobbuf\n")
		newData := make([]byte, cap(bb.data)*2)
		copy(newData, bb.data)
		bb.data = newData
	}
}

// 往bb里面添加字符串，注意填充
func (bb *blobBuf) putString(msgAttr ubusMsgAttr, str string) {
	// 新建一个blobAttr并添加到bb末尾
	var attrLen uint32 = uint32(4 + len(str) + 1)

	roundLen := roundUpLen(attrLen)

	bb.checkSize(roundLen)

	// id, 1 byte
	// len, 3 bytes
	var idLen uint32 = (uint32(msgAttr) << 24) | attrLen

	//fmt.Printf("msgAttr %d, str len %d, attrLen %d, roundLen %d, idLen 0x%x\n", msgAttr, len(str), attrLen, roundLen, idLen)

	// 编码idLen字段
	binary.BigEndian.PutUint32(bb.data[bb.dataLen:], idLen)

	// 编码字符串，data, len(str) + 1 bytes
	copy(bb.data[bb.dataLen+4:], str)

	// pad zero, roundLen - attrLen bytes
	bb.dataLen += roundLen
}

func (bb *blobBuf) putUint32(msgAttr ubusMsgAttr, v uint32) {
	var attrLen uint32 = (4 + 4)

	roundLen := roundUpLen(attrLen)

	bb.checkSize(roundLen)

	var idLen uint32 = (uint32(msgAttr) << 24) | attrLen

	// 编码idLen字段
	binary.BigEndian.PutUint32(bb.data[bb.dataLen:], idLen)

	// 编码无符号整型
	binary.BigEndian.PutUint32(bb.data[bb.dataLen+4:], v)

	bb.dataLen += roundLen
}

// 把b的数据加入到bb中
func (bb *blobBuf) putData(b *blobBuf) {
	var attrLen uint32 = (4 + b.dataLen)

	roundLen := roundUpLen(attrLen)

	bb.checkSize(roundLen)

	var idLen uint32 = (uint32(UBUS_ATTR_DATA) << 24) | attrLen

	binary.BigEndian.PutUint32(bb.data[bb.dataLen:], idLen)

	copy(bb.data[bb.dataLen+4:], b.data)

	bb.dataLen += roundLen
}

type blobAttr struct {
	attrID  uint8
	dataLen uint32
	data    []byte
}

// 解析blobbuf里面的blobAttr
func blobParse(b []byte) ([]*blobAttr, error) {
	ba := make([]*blobAttr, UBUS_ATTR_MAX)

	log.Printf("len(b) = %d\n", len(b))

	var offset uint32
	var id uint8
	var dataLen uint32
	var dataIDLen uint32
	for offset < uint32(len(b)) {
		log.Printf("offset = %d\n", offset)

		dataIDLen = binary.BigEndian.Uint32(b[offset:])
		id = uint8(dataIDLen >> 24 & 0xff)
		dataLen = dataIDLen & 0xffffff

		log.Printf("id = %d, dataLen = %d, roundDataLen = %d\n", id, dataLen, roundUpLen(dataLen))

		if id >= uint8(UBUS_ATTR_MAX) {
			return nil, fmt.Errorf("attr id out of range")
		}

		if ba[id] == nil {
			log.Printf("set ba[%d]\n", id)
			ba[id] = &blobAttr{id, dataLen, b[offset+4 : offset+dataLen]}
		}

		offset += roundUpLen(dataLen)
		log.Printf("next\n")
	}

	return ba, nil
}

func (ba *blobAttr) getUint32() uint32 {
	return binary.BigEndian.Uint32(ba.data)
}
