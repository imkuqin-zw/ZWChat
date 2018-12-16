package net_lib

import (
	"net"
	"sync/atomic"
	"fmt"
	"bufio"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
)

var SessionClosedErr = fmt.Errorf("[session] Session Closed")
var SessionBlockedError = fmt.Errorf("[session] Session Blocked")

var globalSessionId uint64

type SessionCfg struct {
	ReadDeadLine           int //读数据限制的秒数
	WriteDeadLine          int //写数据限制的秒数
	AllowMaxReadBytePerSec int //每秒允许读取最大的字节数
	MaxMessageSize         int //单条消息的最大字节数
}

type Session struct {
	id         uint64 //会话唯一标识
	manager    *Manager
	conn       net.Conn
	r          *bufio.Reader //读取数据的bufer
	codec      Codec         //打包和解包接口
	closeFlag  int32         //链接是否关闭标识, 用int型是为了线程安全的改值
	closeChan  chan int
	sendChan   chan interface{}
	userId     uint64 //用户唯一标识
	msgId      uint64 //消息的唯一标识
	shareKeyId []byte
	shareKey   []byte
	cfg        SessionCfg
}

func NewSession(conn net.Conn, codec Codec, sendChanSize int) *Session {
	return newSession(nil, conn, codec, sendChanSize)
}

func newSession(manager *Manager, conn net.Conn, defaultCode Codec, sendChanSize int) *Session {
	session := &Session{
		id:        atomic.AddUint64(&globalSessionId, 1),
		manager:   manager,
		closeChan: make(chan int),
		conn:      conn,
		r:         bufio.NewReader(conn),
		codec:     defaultCode,
	}
	if sendChanSize > 0 {
		session.sendChan = make(chan interface{}, sendChanSize)
		go session.sendLoop()
	}
	return session
}

func (session *Session) sendLoop() {
	defer session.Close()
	for {
		select {
		case msg := <-session.sendChan:
			buf := session.codec.Packet(msg, session.shareKeyId, session.shareKey)
			if buf != nil {
				if session.Write(buf) != nil {
					return
				}
			}
		case <-session.closeChan:
			return
		}
	}
}

func (session *Session) Close() error {
	if atomic.CompareAndSwapInt32(&session.closeFlag, 0, 1) {
		err := session.conn.Close()
		close(session.closeChan)
		if session.manager != nil {
			session.manager.delSession(session)
		}
		return err
	}
	return SessionClosedErr
}

func (session *Session) Receive() (buf []byte, err error) {
	buf, err = session.codec.UnPack(session.r)
	return
}

func (session *Session) Write(buf []byte) (err error) {
	var onceWriteLen, writtenLen, totalLen = 0, 0, len(buf)
	for writtenLen < totalLen {
		onceWriteLen, err = session.conn.Write(buf[writtenLen:])
		if err != nil {
			logger.Debug("session write: ", zap.Error(err))
			return
		}
		writtenLen += onceWriteLen
	}
	return nil
}

func (session *Session) Send(msg interface{}) error {
	if session.IsClosed() {
		return SessionClosedErr
	}
	if session.sendChan == nil {
		buf := session.codec.Packet(msg, session.shareKeyId, session.shareKey)
		if buf != nil {
			return session.Write(buf)
		}
	}
	select {
	case session.sendChan <- msg:
		return nil
	default:
		return SessionClosedErr
	}
}

func (session *Session) IsClosed() bool {
	return atomic.LoadInt32(&session.closeFlag) == 1
}

func (session *Session) SetUserId(uid uint64) {
	session.userId = uid
}

func (session *Session) GetUserId() uint64 {
	return session.userId
}
