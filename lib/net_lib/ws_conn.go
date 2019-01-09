package net_lib

import (
	"io"
	"io/ioutil"
	"strconv"
	"time"
	"unicode/utf8"
	"errors"
	"bat_common/protobuffs/pbWithClient"
)

const (
	// Frame header byte 0 bits from Section 5.2 of RFC 6455
	finalBit = 1 << 7

	// Frame header byte 1 bits from Section 5.2 of RFC 6455
	maskBit = 1 << 7

	maxFrameHeaderSize         = 2 + 8 + 4 // Fixed header + length + mask
	maxControlFramePayloadSize = 125

	writeWait = time.Second

	defaultReadBufferSize  = 4096
	defaultWriteBufferSize = 4096

	continuationFrame = 0
	noFrame           = -1
)

// Close codes defined in RFC 6455, section 11.7.
const (
	CloseNormalClosure           = 1000
	CloseGoingAway               = 1001
	CloseProtocolError           = 1002
	CloseUnsupportedData         = 1003
	CloseNoStatusReceived        = 1005
	CloseAbnormalClosure         = 1006
	CloseInvalidFramePayloadData = 1007
	ClosePolicyViolation         = 1008
	CloseMessageTooBig           = 1009
	CloseMandatoryExtension      = 1010
	CloseInternalServerErr       = 1011
	CloseServiceRestart          = 1012
	CloseTryAgainLater           = 1013
	CloseTLSHandshake            = 1015
)

// The message types are defined in RFC 6455, section 11.8.
const (
	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage = 1

	// BinaryMessage denotes a binary data message.
	BinaryMessage = 2

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text. Use the FormatCloseMessage
	// function to format a close message payload.
	CloseMessage = 8

	// PingMessage denotes a ping control message. The optional message payload
	// is UTF-8 encoded text.
	PingMessage = 9

	// PongMessage denotes a pong control message. The optional message payload
	// is UTF-8 encoded text.
	PongMessage = 10
)

var (
	errBadWriteOpCode      = errors.New("websocket: bad write message type")
	errWriteClosed         = errors.New("websocket: write closed")
	errInvalidControlFrame = errors.New("websocket: invalid control frame")
)

type WsConn struct {
	readRemaining  int64
	readDecompress bool
}

func isControl(frameType int) bool {
	return frameType == CloseMessage || frameType == PingMessage || frameType == PongMessage
}

// WriteControl writes a control message with the given deadline. The allowed
// message types are CloseMessage, PingMessage and PongMessage.
func (c *Session) WriteControl(messageType int, data []byte, deadline time.Time) error {
	if !isControl(messageType) {
		return errBadWriteOpCode
	}
	if len(data) > maxControlFramePayloadSize {
		return errInvalidControlFrame
	}

	b0 := byte(messageType) | finalBit
	b1 := byte(len(data)) | maskBit

	buf := make([]byte, 0, maxFrameHeaderSize+maxControlFramePayloadSize)
	buf = append(buf, b0, b1)
	buf = append(buf, data...)

	d := time.Hour * 1000
	if !deadline.IsZero() {
		d = deadline.Sub(time.Now())
		if d < 0 {
			return errWriteTimeout
		}
	}

	c.conn.GetConn().SetWriteDeadline(deadline)
	//ws信令消息
	sig := &batprotobuf.Bytes{
		ByteArr: buf,
	}
	msg := CreateWSPlainMessage(sig, c.conn)
	c.conn.SendMessage(&msg)
	//_, err = c.conn.Write(buf)
	//if err != nil {
	//	return c.writeFatal(err)
	//}
	if messageType == CloseMessage {
		c.writeFatal(ErrCloseSent)
	}
	return nil
}

func (c *Session) handleProtocolError(message string) error {
	c.WriteControl(CloseMessage, FormatCloseMessage(CloseProtocolError, message), time.Now().Add(writeWait))
	return errors.New("websocket: " + message)
}

func (c *Session) advanceFrame() (int, error) {
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
	c.readRemaining = int64(p[1] & 0x7f)

	switch frameType {
	case CloseMessage, PingMessage, PongMessage:
		if c.readRemaining > maxControlFramePayloadSize {
			return noFrame, c.handleProtocolError("control frame length > 125")
		}
		if !final {
			return noFrame, c.handleProtocolError("control frame not final")
		}
	case TextMessage, BinaryMessage:
		if !c.readFinal {
			return noFrame, c.handleProtocolError("message start before final message frame")
		}
		c.readFinal = final
	case continuationFrame:
		if c.readFinal {
			return noFrame, c.handleProtocolError("continuation after final message frame")
		}
		c.readFinal = final
	default:
		return noFrame, c.handleProtocolError("unknown opcode " + strconv.Itoa(frameType))
	}

	// 3. Read and parse frame length.

	switch c.readRemaining {
	case 126:
		p, err := c.read(2)
		if err != nil {
			return noFrame, err
		}
		//16位的数据包大小
		c.readRemaining = int64(binary.BigEndian.Uint16(p))
	case 127:
		p, err := c.read(8)
		if err != nil {
			return noFrame, err
		}
		//64位的数据包大小
		c.readRemaining = int64(binary.BigEndian.Uint64(p))
	}

	// 4. Handle frame masking.
	if mask != c.isServer {
		return noFrame, c.handleProtocolError("incorrect mask flag")
	}

	if mask {
		c.readMaskPos = 0
		//4字节的mask key
		p, err := c.read(len(c.readMaskKey))
		if err != nil {
			return noFrame, err
		}
		copy(c.readMaskKey[:], p)
	}
	// 5. For text and binary messages, enforce read limit and return.
	//如果是文本和二进制消息，检测读取限制，如果超过了则强制退出
	if frameType == continuationFrame || frameType == TextMessage || frameType == BinaryMessage {
		c.readLength += c.readRemaining
		if c.readLimit > 0 && c.readLength > c.readLimit {
			c.WriteControl(CloseMessage, FormatCloseMessage(CloseMessageTooBig, ""), time.Now().Add(writeWait))
			return noFrame, ErrReadLimit
		}
		return frameType, nil
	}

	// 6. Read control frame payload.
	var payload []byte
	if c.readRemaining > 0 {
		payload, err = c.read(int(c.readRemaining))
		c.readRemaining = 0
		if err != nil {
			return noFrame, err
		}
		if c.isServer {
			//解码
			maskBytes(c.readMaskKey, 0, payload)
		}
	}

	// 7. Process control frame payload.

	switch frameType {
	case PongMessage:
		if err := c.handlePong(string(payload)); err != nil {
			return noFrame, err
		}
	case PingMessage:
		if err := c.handlePing(string(payload)); err != nil {
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
		if err := c.handleClose(closeCode, closeText); err != nil {
			return noFrame, err
		}
		return noFrame, &CloseError{Code: closeCode, Text: closeText}
	}

	return frameType, nil
}
