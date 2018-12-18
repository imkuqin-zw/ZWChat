package net_lib

import (
	"encoding/base64"
	"github.com/golang/protobuf/proto"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"github.com/kataras/go-errors"
	"go.uber.org/zap"
)

var DefaultCode = ProtoCodeInstance
var ProtoCodeInstance = new(ProtoTcpCode)
var DataLenErr = errors.New("receive data length error")

type Codec interface {
	Packet(src interface{}, shareKeyId, shareKey []byte) ([]byte, error)
	UnPack(session *Session) ([]byte, error)
}

type ProtoTcpCode struct{}

func (codec *ProtoTcpCode) Packet(msg interface{}, authKeyId, shareKey []byte) ([]byte, error) {
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
		msgKey, enBytes, err := codec.encrypt(shareKey, body)
		if err != nil {
			return nil, err
		}
		result.WriteUint32(24 + uint32(len(enBytes)))
		result.Write(authKeyId, msgKey, enBytes)
	}
	return result.Bytes(), nil
}

func (codec *ProtoTcpCode) UnPack(session *Session) ([]byte, error) {
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
	var result = data
	if msgKey != nil {
		shareKey := session.GetShareKey(authKey)
		result, err = codec.decrypt(shareKey, msgKey, data)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (codec *ProtoTcpCode) encrypt(shareKey, data []byte) ([]byte, []byte, error) {
	var key, iv []byte
	msgKey := DeriveMsgKey(shareKey, data)
	DeriveAESKey(shareKey, msgKey, key, iv)
	logger.Debug("Proto Packet: ",
		zap.String("msgKey", base64.StdEncoding.EncodeToString(msgKey)),
		zap.String("AESKey", base64.StdEncoding.EncodeToString(key)),
		zap.String("IV", base64.StdEncoding.EncodeToString(iv)))
	enBytes, err := AESCBCPadEncrypt(nil, data, key, iv)
	if err != nil {
		logger.Error("Proto Packet encrypt err: ", zap.Error(err))
		return nil, nil, err
	}
	return msgKey, enBytes, nil
}

func (Codec *ProtoTcpCode) decrypt(shareKey, msgKey, data []byte) ([]byte, error) {
	var key, iv []byte
	DeriveAESKey(shareKey, msgKey, key, iv)
	result, err := AESCBCDecrypt(nil, data, key, iv)
	if err != nil {
		logger.Error("Proto decrypt err: ", zap.Error(err))
		return nil, err
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
