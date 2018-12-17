package net_lib

import (
	"encoding/base64"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/henrylee2cn/pholcus/common/session"
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

func (codec *ProtoTcpCode) UnPack(session *Session) ([]byte, error) {
	length, err := codec.getDataLen(session)
	if err != nil {
		logger.Error("Proto UnPack get length err: ", zap.Error(err))
		return nil, err
	}
	if length <= uint32(28) || length < session.cfg.MaxMessageSize {
		logger.Error("Proto UnPack length error:", zap.Uint32("length", length))
		return nil, DataLenErr
	}
	if err = codec.getAuthKeyId(session); err != nil {
		return nil, err
	}

	msgKey, err := codec.getMsgKey(session)
	if err != nil {
		return nil, err
	}
	if msgKey == nil {

	} else {

	}
	//buf := make([]byte, length)
	//if _, err = session.r.Read(buf[0:length]); err != nil {
	//	logger.Error("Proto UnPack get data err: ", zap.Error(err))
	//	return nil, err
	//}

	return buf, nil
}

func (Codec *ProtoTcpCode) getData()

func (codec *ProtoTcpCode) getAuthKeyId(session *Session) error {
	buf := make([]byte, 8)
	if _, err := session.r.Read(buf[0:8]); err != nil {
		logger.Error("Proto getAuthKeyId err: ", zap.Error(err))
		return err
	}
	if !IsBytesAllZero(buf) {
		session.shareKey = buf
	}
	return nil
}

func (codec *ProtoTcpCode) getMsgKey(session *Session) ([]byte, error) {
	buf := make([]byte, 16)
	if _, err := session.r.Read(buf[0:16]); err != nil {
		logger.Error("Proto getAuthKeyId err: ", zap.Error(err))
		return nil, err
	}
	if !IsBytesAllZero(buf) {
		return buf, nil
	}
	return nil, nil
}

func (codec *ProtoTcpCode) getDataLen(session *Session) (uint32, error) {
	var buf = make([]byte, 4)
	_, err := session.r.Read(buf[0:4])
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf), nil
}
