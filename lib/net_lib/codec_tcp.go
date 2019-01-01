package net_lib

import (
	"time"
	"github.com/golang/protobuf/proto"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
)

type ProtoTcpCode struct{}

func (codec *ProtoTcpCode) Packet(msg interface{}, session *Session) ([]byte, error) {
	authKeyId := session.GetShareKeyId()
	shareKey := session.GetShareKey(authKeyId)
	body, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		logger.Error("Proto Packet Marshal err: ", zap.Error(err))
		return nil, err
	}
	result := new(Writer)
	if len(shareKey) == 0 { // 不加密
		result.WriteUint32(24 + uint32(len(body)))
		result.Write(authKeyId, make([]byte, 16), body)
	} else { // 加密
		msgKey, enBytes, err := encrypt(shareKey, body)
		if err != nil {
			return nil, err
		}
		result.WriteUint32(24 + uint32(len(enBytes)))
		result.Write(authKeyId, msgKey, enBytes)
	}
	return result.Bytes(), nil
}

func (codec *ProtoTcpCode) UnPack(session *Session) ([]byte, error) {
	if session.cfg.ReadDeadLine > 0 {
		deadTime := time.Now().Add(time.Second * time.Duration(session.cfg.ReadDeadLine))
		session.conn.SetReadDeadline(deadTime)
	}
	length, err := codec.getDataLen(session.r)
	if err != nil {
		logger.Error("Proto UnPack getDataLen err: ", zap.Error(err))
		return nil, err
	}
	if length <= 28 || length < session.cfg.MaxMsgSize {
		logger.Error("Proto UnPack length error:", zap.Uint32("length", length))
		return nil, DataLenErr
	}
	authKey, err := codec.getAuthKeyId(session)
	if err != nil {
		return nil, err
	}
	msgKey, err := codec.getMsgKey(session.r)
	if err != nil {
		return nil, err
	}
	data, err := codec.getData(session.r, length-24)
	if err != nil {
		return nil, err
	}
	if session.cfg.ReadDeadLine > 0 {
		session.conn.SetReadDeadline(time.Time{})
	}
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

func (codec *ProtoTcpCode) getData(r *Reader, length uint32) ([]byte, error) {
	buf, err := r.ReadN(length)
	if err != nil {
		logger.Error("Proto getData err: ", zap.Error(err))
		return nil, err
	}
	return buf, nil
}

func (codec *ProtoTcpCode) getAuthKeyId(session *Session) ([]byte, error) {
	buf, err := session.r.ReadN(8)
	if err != nil {
		logger.Error("Proto getAuthKeyId err: ", zap.Error(err))
		return nil, err
	}
	if !IsBytesAllZero(buf) {
		session.SetShareKeyId(buf)
		return buf, nil
	}
	return nil, nil
}

func (codec *ProtoTcpCode) getMsgKey(r *Reader) ([]byte, error) {
	buf, err := r.ReadN(16)
	if err != nil {
		logger.Error("Proto getAuthKeyId err: ", zap.Error(err))
		return nil, err
	}
	if !IsBytesAllZero(buf) {
		return buf, nil
	}
	return nil, nil
}

func (codec *ProtoTcpCode) getDataLen(r *Reader) (uint32, error) {
	return r.ReadUint32()
}

