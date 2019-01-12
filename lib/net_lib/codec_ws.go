package net_lib

import (
	"github.com/golang/protobuf/proto"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"time"
	"fmt"
)

type ProtoWsCode struct{}

func (codec *ProtoWsCode) Packet(msg interface{}, session *Session) ([]byte, error) {
	if data, ok := msg.([]byte); ok {
		return data, nil
	}
	authKeyId := session.GetShareKeyId()
	shareKey := session.GetShareKey(authKeyId)
	body, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		logger.Error("Proto Packet Marshal err: ", zap.Error(err))
		return nil, err
	}
	extral := new(Writer)
	if len(shareKey) == 0 {
		extral.Write(authKeyId, make([]byte, 16), body)
	}
	if len(shareKey) != 0 {
		msgKey, enBytes, err := encrypt(shareKey, body)
		if err != nil {
			return nil, err
		}
		extral.Write(authKeyId, msgKey, enBytes)
	}
	return session.flushFrame(extral.Bytes()), nil
}

func (codec *ProtoWsCode) UnPack(session *Session) ([]byte, error) {
	if session.cfg.ReadDeadLine > 0 {
		deadTime := time.Now().Add(time.Second * time.Duration(session.cfg.ReadDeadLine))
		session.conn.SetReadDeadline(deadTime)
	}
	if session.cfg.ReadDeadLine > 0 {
		session.conn.SetReadDeadline(time.Time{})
	}
	for session.wsConn.readErr == nil {
		frameType, err := session.advanceFrame()
		if err != nil {
			session.wsConn.readErr = err
			break
		}
		fmt.Println("dsf")
		if frameType != TextMessage && frameType != BinaryMessage {
			continue
		}
		var reader io.Reader
		reader = NewMessageReader(session)
		data, err := ioutil.ReadAll(reader)
		if err != nil {
			session.wsConn.readErr = err
			break
		}
		if session.cfg.ReadDeadLine > 0 {
			session.conn.SetReadDeadline(time.Time{})
		}
		authKey := codec.getAuthKeyId(data, session)
		msgKey := codec.getMsgKey(data)
		var result = data
		if msgKey != nil {
			shareKey := session.GetShareKey(authKey)
			result, err = decrypt(shareKey, msgKey, data)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	return nil, session.wsConn.readErr
}

func (codec *ProtoWsCode) getAuthKeyId(data []byte, session *Session) []byte {
	buf, data := data[:8], data[8:]
	if !IsBytesAllZero(buf) {
		session.SetShareKeyId(buf)
		return buf
	}
	return nil
}

func (codec *ProtoWsCode) getMsgKey(data []byte) []byte {
	buf, data := data[:16], data[16:]
	if !IsBytesAllZero(buf) {
		return buf
	}
	return nil
}


