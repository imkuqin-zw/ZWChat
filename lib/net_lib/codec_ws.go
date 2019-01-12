package net_lib

import (
	"github.com/golang/protobuf/proto"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"time"
	"bat_common/netlib/gennetwork"
	"encoding/binary"
	"unicode/utf8"
	"strconv"
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
	return codec.flushFrame(extral.Bytes()), nil
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
		frameType, err := codec.advanceFrame(session)
		if err != nil {
			session.wsConn.readErr = err
			break
		}
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

func (codec *ProtoWsCode) flushFrame(extra []byte) []byte {
	length := len(extra)
	b0 := byte(gennetwork.TextMessage) | finalBit
	b1 := byte(0)
	result := make([]byte, 0, maxFrameHeaderSize + length)
	copy(result[maxFrameHeaderSize:], extra)
	switch {
	case length >= 65536:
		result[0] = b0
		result[1] = b1 | 127
		binary.BigEndian.PutUint64(result[2:], uint64(length))
	case length > 125:
		result[6] = b0
		result[7] = b1 | 126
		binary.BigEndian.PutUint16(result[2:], uint16(length))
	default:
		result[8] = b0
		result[9] = b1 | byte(length)
	}
	return result
}

func (codec *ProtoWsCode) advanceFrame(c *Session) (int, error) {
	// 1. Skip remainder of previous frame.
	//读取上一次剩余的帧
	if c.wsConn.readRemaining > 0 {
		if _, err := io.CopyN(ioutil.Discard, c.r, c.wsConn.readRemaining); err != nil {
			return noFrame, err
		}
	}

	// 2. Read and parse first two bytes of frame header.
	// 读取头两个字节的头
	p, err := c.r.ReadN(2)
	if err != nil {
		return noFrame, err
	}
	//第1位表示是否最后一个字节，1表示最后一帧
	final := p[0]&finalBit != 0
	//opcode消息类型
	//%x0 代表一个继续帧
	//%x1 代表一个文本帧
	//%x2 代表一个二进制帧
	//%x3-7 保留用于未来的非控制帧
	//%x8 代表连接关闭
	//%x9 代表ping
	//%xA 代表pong
	//%xB-F 保留用于未来的控制帧
	frameType := int(p[0] & 0xf)
	//是否有掩码
	mask := p[1]&maskBit != 0
	//当前帧剩余位（消息长度）
	c.wsConn.readRemaining = int64(p[1] & 0x7f)

	switch frameType {
	case CloseMessage, PingMessage, PongMessage:
		if c.wsConn.readRemaining > maxControlFramePayloadSize {
			return noFrame, c.handleProtocolError("control frame length > 125")
		}
		if !final {
			return noFrame, c.handleProtocolError("control frame not final")
		}
	case TextMessage, BinaryMessage:
		if !c.wsConn.readFinal {
			return noFrame, c.handleProtocolError("message start before final message frame")
		}
		c.wsConn.readFinal = final
	case continuationFrame:
		if c.wsConn.readFinal {
			return noFrame, c.handleProtocolError("continuation after final message frame")
		}
		c.wsConn.readFinal = final
	default:
		return noFrame, c.handleProtocolError("unknown opcode " + strconv.Itoa(frameType))
	}

	// 3. Read and parse frame length.
	switch c.wsConn.readRemaining {
	case 126:
		p, err := c.r.ReadN(2)
		if err != nil {
			return noFrame, err
		}
		//16位的数据包大小
		c.wsConn.readRemaining = int64(binary.BigEndian.Uint16(p))
	case 127:
		p, err := c.r.ReadN(8)
		if err != nil {
			return noFrame, err
		}
		//64位的数据包大小
		c.wsConn.readRemaining = int64(binary.BigEndian.Uint64(p))
	}

	if mask {
		c.wsConn.readMaskPos = 0
		//4字节的mask key
		p, err := c.r.ReadN(len(c.wsConn.readMaskKey))
		if err != nil {
			return noFrame, err
		}
		copy(c.wsConn.readMaskKey[:], p)
	}
	// 5. For text and binary messages, enforce read limit and return.
	//如果是文本和二进制消息，检测读取限制，如果超过了则强制退出
	if frameType == continuationFrame || frameType == TextMessage || frameType == BinaryMessage {
		c.wsConn.readLength += uint32(c.wsConn.readRemaining)
		if c.cfg.MaxMsgSize > 0 && c.wsConn.readLength > c.cfg.MaxMsgSize {
			logger.Debug("advanceFrame lenth error:", zap.Uint32("length", c.wsConn.readLength))
			c.WriteControl(CloseMessage, FormatCloseMessage(CloseMessageTooBig, ""), time.Now().Add(writeWait))
			return noFrame, DataLenErr
		}
		return frameType, nil
	}

	// 6. Read control frame payload.
	var payload []byte
	if c.wsConn.readRemaining > 0 {
		payload, err = c.r.ReadN(int(c.wsConn.readRemaining))
		c.wsConn.readRemaining = 0
		if err != nil {
			return noFrame, err
		}
		//解码
		maskBytes(c.wsConn.readMaskKey, 0, payload)
	}

	// 7. Process control frame payload.
	switch frameType {
	case PongMessage:
		if err := c.wsConn.handlePong(payload); err != nil {
			return noFrame, err
		}
	case PingMessage:
		if err := c.wsConn.handlePing(payload); err != nil {
			return noFrame, err
		}
	case CloseMessage:
		closeCode := CloseNoStatusReceived
		closeText := ""
		if len(payload) >= 2 {
			closeCode = int(binary.BigEndian.Uint16(payload))
			if !isValidReceivedCloseCode(closeCode) {
				return noFrame, c.handleProtocolError("invalid close code")
			}
			closeText = string(payload[2:])
			if !utf8.ValidString(closeText) {
				return noFrame, c.handleProtocolError("invalid utf8 payload in close frame")
			}
		}
		return noFrame, errClientClose
	}

	return frameType, nil
}


