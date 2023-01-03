package tcp_client

import (
	"bytes"
	"encoding/binary"
)

var defaultHeaderLen = uint32(9)

// Message 消息
type Message struct {
	DataLen uint32 //消息的长度
	Type    uint32 //消息的ID
	Seq     uint8  //消息序号
	Data    []byte //消息的内容
}

// NewMsgPackage 创建一个Message消息包
func NewMsgPackage(Type uint32, Seq uint8, data []byte) *Message {
	return &Message{
		DataLen: uint32(len(data)),
		Type:    Type,
		Data:    data,
		Seq:     Seq,
	}
}

// GetDataLen 获取消息数据段长度
func (msg *Message) GetDataLen() uint32 {
	return msg.DataLen
}

// GetMsgID 获取消息ID
func (msg *Message) GetMsgType() uint32 {
	return msg.Type
}

// GetData 获取消息内容
func (msg *Message) GetData() []byte {
	return msg.Data
}

// GetData 获取消息序号
func (msg *Message) GetSeq() uint8 {
	return msg.Seq
}

// SetDataLen 设置消息数据段长度
func (msg *Message) SetDataLen(len uint32) {
	msg.DataLen = len
}

// SetMsgID 设计消息ID
func (msg *Message) SetMsgID(msgID uint32) {
	msg.Type = msgID
}

// SetData 设计消息内容
func (msg *Message) SetData(data []byte) {
	msg.Data = data
}

// GetHeadLen 获取包头长度方法
func (this *Message) GetHeadLen() uint32 {
	//Type uint32(4字节) +  DataLen uint32(4字节) + Seq uint8 (1字节)
	return defaultHeaderLen
}

// Pack 封包方法(压缩数据)
func (this *Message) Pack() ([]byte, error) {
	//创建一个存放bytes字节的缓冲
	dataBuff := bytes.NewBuffer([]byte{})

	//写dataLen
	if err := binary.Write(dataBuff, binary.LittleEndian, this.GetDataLen()); err != nil {
		return nil, err
	}

	//写msgTye
	if err := binary.Write(dataBuff, binary.LittleEndian, this.GetMsgType()); err != nil {
		return nil, err
	}

	//写Seq
	if err := binary.Write(dataBuff, binary.LittleEndian, this.GetSeq()); err != nil {
		return nil, err
	}

	//写data数据
	if err := binary.Write(dataBuff, binary.LittleEndian, this.GetData()); err != nil {
		return nil, err
	}

	return dataBuff.Bytes(), nil
}

// Unpack 拆包方法(解压数据)
func (this *Message) Unpack(binaryData []byte) error {
	//创建一个从输入二进制数据的ioReader
	dataBuff := bytes.NewReader(binaryData)

	//读dataLen
	if err := binary.Read(dataBuff, binary.LittleEndian, &this.DataLen); err != nil {
		return err
	}

	//读msgID
	if err := binary.Read(dataBuff, binary.LittleEndian, &this.Type); err != nil {
		return err
	}

	//读消息序号
	if err := binary.Read(dataBuff, binary.LittleEndian, &this.Seq); err != nil {
		return err
	}

	//这里只需要把head的数据拆包出来就可以了，然后再通过head的长度，再从conn读取一次数据
	return nil
}
