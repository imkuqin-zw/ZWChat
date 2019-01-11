package net_lib

import "github.com/golang/protobuf/proto"

//MTGeNetMessage 通用消息
type OutMessage struct {
	BatId        uint32
	MessageKey   [16]byte
	ServerSalt   uint64
	SessionID    uint64
	MessageID    uint64
	AckId        uint64
	MessageSeqNo uint32
	MessageLen   uint32
	MessageObj   proto.Message
	IsAck        bool
	//发送时间戳
	SndTime int64
	//UserID
	UserID uint32
	//全球唯一的消息ID
	GMessageID uint64
	DCID       int64
	CrcId      uint32
	Err        error
	WsSignal   bool
}
