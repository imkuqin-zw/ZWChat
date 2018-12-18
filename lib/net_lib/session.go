package net_lib

import (
	"bufio"
	"fmt"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"net"
	"sync/atomic"
	"time"
)

var SessionClosedErr = fmt.Errorf("[session] Session Closed")
var SessionBlockedError = fmt.Errorf("[session] Session Blocked")

var globalSessionId uint64

type SessionCfg struct {
	ReadDeadLine  int //读数据限制的秒数
	WriteDeadLine int //写数据限制的秒数
	maxAttempts   int //最大限制数量
	duration      int64
	interval      int64  //窗口时间间隔(s)
	count         int64  //窗口数量
	MaxMsgSize    uint32 //单条消息的最大字节数
}

type Session struct {
	id         uint64 //会话唯一标识
	manager    *Manager
	conn       net.Conn
	r          *Reader //读取数据的bufer
	codec      Codec   //打包和解包接口
	closeFlag  int32   //链接是否关闭标识, 用int型是为了线程安全的改值
	closeChan  chan int
	sendChan   chan interface{}
	userId     uint64 //用户唯一标识
	msgId      uint64 //消息的唯一标识
	shareKeyId []byte
	shareKey   []byte
	cfg        SessionCfg
}

//func NewSession(conn net.Conn, codec Codec, sendChanSize int, cfg SessionCfg) *Session {
//	return newSession(nil, conn, codec, sendChanSize, cfg)
//}

func newSession(manager *Manager, conn net.Conn, defaultCode Codec, sendChanSize int, cfg SessionCfg) *Session {
	session := &Session{
		id:        atomic.AddUint64(&globalSessionId, 1),
		manager:   manager,
		closeChan: make(chan int),
		conn:      conn,
		r:         NewReader(bufio.NewReader(conn)),
		codec:     defaultCode,
		cfg:       cfg,
	}
	if sendChanSize > 0 {
		session.sendChan = make(chan interface{}, sendChanSize)
		go session.sendLoop()
	}
	return session
}

func (session *Session) SetShareKeyId(shareKeyId []byte) {
	session.shareKeyId = shareKeyId
}

func (session *Session) GetShareKeyId() []byte {
	return session.shareKeyId
}

func (session *Session) SetShareKey(shareKey []byte) {
	session.shareKeyId = shareKey
}

func (session *Session) GetShareKey(shareKeyId []byte) []byte {
	if !IsBytesAllZero(session.shareKey) {
		return session.shareKey
	} else {

	}
}

func (session *Session) sendLoop() {
	defer session.Close()
	for {
		select {
		case msg := <-session.sendChan:
			buf, err := session.codec.Packet(msg, session.shareKeyId, session.shareKey)
			if err != nil {
				return
			}
			if session.cfg.WriteDeadLine > 0 {
				deadTime := time.Now().Add(time.Second * time.Duration(session.cfg.WriteDeadLine))
				session.conn.SetWriteDeadline(deadTime)
			}
			if err = session.Write(buf); err != nil {
				logger.Error("session.Write error: ", zap.Error(err))
				return
			}
			if session.cfg.WriteDeadLine > 0 {
				session.conn.SetWriteDeadline(time.Time{})
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
	if session.cfg.ReadDeadLine > 0 {
		deadTime := time.Now().Add(time.Second * time.Duration(session.cfg.ReadDeadLine))
		session.conn.SetReadDeadline(deadTime)
	}
	buf, err = session.codec.UnPack(session)
	if err != nil {
		return nil, err
	}
	if session.cfg.ReadDeadLine > 0 {
		session.conn.SetReadDeadline(time.Time{})
	}
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
		buf, err := session.codec.Packet(msg, session.shareKeyId, session.shareKey)
		if err != nil {
			return err
		}
		if err = session.Write(buf); err != nil {
			logger.Error("session.Write error: ", zap.Error(err))
			return err
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
