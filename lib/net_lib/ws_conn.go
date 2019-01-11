package net_lib

import (
	"bat_common/protobuffs/pbWithClient"
	"encoding/binary"
	"errors"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"strconv"
	"time"
	"unicode/utf8"
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

var validReceivedCloseCodes = map[int]bool{
	// see http://www.iana.org/assignments/websocket/websocket.xhtml#close-code-number

	CloseNormalClosure:           true,
	CloseGoingAway:               true,
	CloseProtocolError:           true,
	CloseUnsupportedData:         true,
	CloseNoStatusReceived:        false,
	CloseAbnormalClosure:         false,
	CloseInvalidFramePayloadData: true,
	ClosePolicyViolation:         true,
	CloseMessageTooBig:           true,
	CloseMandatoryExtension:      true,
	CloseInternalServerErr:       true,
	CloseServiceRestart:          true,
	CloseTryAgainLater:           true,
	CloseTLSHandshake:            false,
}

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
	errWriteTimeout        = errors.New("websocket: write timeout")
	errBadWriteOpCode      = errors.New("websocket: bad write message type")
	errWriteClosed         = errors.New("websocket: write closed")
	errInvalidControlFrame = errors.New("websocket: invalid control frame")
	errClientClose         = errors.New("websocket: client close")
	errUnexpectedEOF       = errors.New("websocket: unexpected EOF")
)

type WsConn struct {
	readRemaining  int64
	readDecompress bool
	readFinal      bool
	readLength     uint32
	readMaskPos    int
	readMaskKey    [4]byte
	readErr        error
	handlePong     func([]byte) error
	handlePing     func([]byte) error
	handleClose    func(int, string) error
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
	if messageType == CloseMessage {
		c.SetWaite()
	}
	c.conn.SetWriteDeadline(deadline)
	//ws信令消息
	sig := &batprotobuf.Bytes{
		ByteArr: buf,
	}
	msg := CreateWSPlainMessage(sig, c.conn)
	c.Send(msg)
	return nil
}

// FormatCloseMessage formats closeCode and text as a WebSocket close message.
func FormatCloseMessage(closeCode int, text string) []byte {
	buf := make([]byte, 2+len(text))
	binary.BigEndian.PutUint16(buf, uint16(closeCode))
	copy(buf[2:], text)
	return buf
}

func isValidReceivedCloseCode(code int) bool {
	return validReceivedCloseCodes[code] || (code >= 3000 && code <= 4999)
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

func NewMessageReader(session *Session) io.Reader {
	return &messageReader{session: session}
}

type messageReader struct{ session *Session }

func (r *messageReader) Read(b []byte) (int, error) {
	s := r.session
	for s.wsConn.readErr == nil {
		if s.wsConn.readRemaining > 0 {
			if int64(len(b)) > s.wsConn.readRemaining {
				b = b[:s.wsConn.readRemaining]
			}
			n, err := s.r.Read(b)
			s.wsConn.readErr = err
			s.wsConn.readMaskPos = maskBytes(s.wsConn.readMaskKey, s.wsConn.readMaskPos, b[:n])
			s.wsConn.readRemaining -= int64(n)
			if s.wsConn.readRemaining > 0 && s.wsConn.readErr == io.EOF {
				s.wsConn.readErr = errUnexpectedEOF
			}
			return n, s.wsConn.readErr
		}

		if s.wsConn.readFinal {
			return 0, io.EOF
		}

		frameType, err := s.advanceFrame()
		switch {
		case err != nil:
			s.wsConn.readErr = err
		case frameType == TextMessage || frameType == BinaryMessage:
			s.wsConn.readErr = errors.New("websocket: internal error, unexpected text or binary in Reader")
		}
	}

	err := s.wsConn.readErr
	if err == io.EOF {
		err = errUnexpectedEOF
	}
	return 0, err
}
