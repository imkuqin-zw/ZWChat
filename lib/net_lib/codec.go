package net_lib

import (
	"net"
	"github.com/micro/protobuf/proto"
)

type Codec interface{
	Packet(src interface{}) ([]byte, error)
	UnPack(conn net.Conn) ([]byte, error)
}

type ProtobufCodec struct {

}

func (codec *ProtobufCodec) Packet(msg interface{}) ([]byte, error) {
	body, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		return nil, err
	}
	lens := len(body)
	buf := make([]byte, 5 + lens)
	buf[0] = '0'
	buf[1] = byte(uint32(lens))
	buf[2] = byte(uint32(lens) >> 8)
	buf[3] = byte(uint32(lens) >> 16)
	buf[4] = byte(uint32(lens) >> 24)
	copy(buf[5:], msg.([]byte))
	return buf, nil
}

func (codec *ProtobufCodec) UnPack(conn net.Conn) ([]byte, error) {
	var buf = make([]byte,5)
	_, err := conn.Read(buf[0:5])
	if err != nil {
		return nil, err
	}
	lens := int(uint32(buf[1]) | uint32(buf[2])<<8 | uint32(buf[3])<<16 | uint32(buf[4])<<24)
	buf = make([]byte, lens)
	_, err = conn.Read(buf[0:lens])
	if err != nil {
		return nil, err
	}
	return buf, nil
}