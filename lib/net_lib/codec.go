package net_lib

import (
	"encoding/base64"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"errors"
)

var ProtoTcp = new(ProtoTcpCode)
var ProtoHttp = new(ProtoHttpCode)
var ProtoWs = new(ProtoWsCode)
var DataLenErr = errors.New("receive data length error")

type Codec interface {
	Packet(src interface{}, session *Session) ([]byte, error)
	UnPack(session *Session) ([]byte, error)
}

func encrypt(shareKey, data []byte) ([]byte, []byte, error) {
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

func decrypt(shareKey, msgKey, data []byte) ([]byte, error) {
	var key, iv []byte
	DeriveAESKey(shareKey, msgKey, key, iv)
	result, err := AESCBCDecrypt(nil, data, key, iv)
	if err != nil {
		logger.Error("Proto decrypt err: ", zap.Error(err))
		return nil, err
	}
	return result, nil
}
