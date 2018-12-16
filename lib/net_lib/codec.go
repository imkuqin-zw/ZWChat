package net_lib

import (
	"github.com/golang/protobuf/proto"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"encoding/base64"
	"bufio"
)

var DefaultCode = ProtoCodeInstance

var ProtoCodeInstance = new(ProtoTcpCode)

type Codec interface {
	Packet(src interface{}, shareKeyId, shareKey []byte) ([]byte)
	UnPack(conn *bufio.Reader) ([]byte, error)
}

type ProtoTcpCode struct{}

func (codec *ProtoTcpCode) Packet(msg interface{}, authKeyId, shareKey []byte) ([]byte) {
	body, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		logger.Error("Proto Packet Marshal err: ", zap.Error(err))
		return nil
	}
	result := new(Writer)
	if len(shareKey) == 0 { // 不加密
		result.WriteUint32(24 + uint32(len(body)))
		result.Write(authKeyId, make([]byte, 16), body)
	} else { // 加密
		var key, iv []byte
		msgKey := DeriveMsgKey(shareKey, body)
		logger.Debug("Proto Packet: ", zap.String("msgKey", base64.StdEncoding.EncodeToString(msgKey)))
		DeriveAESKey(shareKey, msgKey, key, iv)
		logger.Debug("Proto Packet: ", zap.String("AESKey", base64.StdEncoding.EncodeToString(key)))
		logger.Debug("Proto Packet: ", zap.String("IV", base64.StdEncoding.EncodeToString(iv)))
		enBytes, err := AESCBCPadEncrypt(nil, body, key, iv)
		if err != nil {
			logger.Error("Proto Packet encrypt err: ", zap.Error(err))
			return nil
		}
		result.WriteUint32(24 + uint32(len(enBytes)))
		result.Write(authKeyId, msgKey, enBytes)
	}
	return result.Bytes()
}

func (codec *ProtoTcpCode) UnPack(conn *bufio.Reader) ([]byte) {
	var buf = make([]byte, 4)
	_, err := conn.Read(buf[0:4])
	if err != nil {
		logger.Error("Proto UnPack getLen err: ", zap.Error(err))
		return nil
	}
	lens := int(uint32(buf[1]) | uint32(buf[2])<<8 | uint32(buf[3])<<16 | uint32(buf[4])<<24)
	buf = make([]byte, lens)
	_, err = conn.Read(buf[0:lens])
	if err != nil {
		return nil, err
	}
	return buf, nil
}
