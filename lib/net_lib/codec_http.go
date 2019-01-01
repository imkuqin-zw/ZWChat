package net_lib

import (
	"fmt"
	"time"
	"net/http"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"io/ioutil"
	"github.com/golang/protobuf/proto"
)

type ProtoHttpCode struct{}

func (codec *ProtoHttpCode) Packet(msg interface{}, session *Session) ([]byte, error) {
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
	header := new(Writer)
	header.WriteStrings("HTTP/1.1 200 OK\r\n")
	header.WriteStrings("Content-Type: text/plain\r\n")
	header.WriteStrings("Connection: Keep-Alive\r\n")
	header.WriteStrings(fmt.Sprintf("Content-Length: %d\r\n", result.Len()))
	TimeFormat := "Mon, 02 Jan 2006 15:04:05 GMT"
	dataStr := time.Now().UTC().Format(TimeFormat)
	header.WriteStrings(fmt.Sprintf("Date:%s\r\n\r\n", dataStr))
	result.Write(header.Bytes())
	return result.Bytes(), nil
}

func (codec *ProtoHttpCode) UnPack(session *Session) ([]byte, error) {
	if session.cfg.ReadDeadLine > 0 {
		deadTime := time.Now().Add(time.Second * time.Duration(session.cfg.ReadDeadLine))
		session.conn.SetReadDeadline(deadTime)
	}
	r, err := http.ReadRequest(session.r.r)
	if err != nil {
		logger.Error("ProtoHttpCode UnPack ReadRequest err: ", zap.Error(err))
		return nil, err
	}
	defer r.Body.Close()
	if session.cfg.ReadDeadLine > 0 {
		session.conn.SetReadDeadline(time.Time{})
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(body) < 24 {
		logger.Error("ProtoHttpCode UnPack err: ", zap.Error(DataLenErr))
		return nil, DataLenErr
	}
	authKey := body[:8]
	if !IsBytesAllZero(authKey) {
		session.SetShareKeyId(authKey)
	}
	msgKey := body[9:25]
	data := body[25:]
	result := data
	if msgKey != nil {
		shareKey := session.GetShareKey(authKey)
		result, err = decrypt(shareKey, msgKey, data)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

