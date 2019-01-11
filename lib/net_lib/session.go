package net_lib

import (
	"bufio"
	"fmt"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var SessionClosedErr = fmt.Errorf("[session] Session Closed")
var SessionBlockedErr = fmt.Errorf("[session] Session Blocked")
var SessionWaitingErr = fmt.Errorf("[session] Session Waiting")

var globalSessionId uint64

const (
	TCP = iota
	HTTP
	WS
)

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
	r          *Reader        //读取数据的bufer
	codec      Codec          //打包和解包接口
	waitFlag   int32          //等待关闭的状态（不接受消息）
	closeWait  sync.WaitGroup //等待关闭
	closeFlag  int32          //连接是否关闭标识, 用int型是为了线程安全的改值
	closeChan  chan int
	sendChan   chan *OutMessage
	userId     uint64 //用户唯一标识
	msgId      uint64 //消息的唯一标识
	shareKeyId []byte
	shareKey   []byte
	cfg        SessionCfg
	connType   int8 //连接类型
	wsConn     WsConn
}

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

//设置连接处于等待状态
func (session *Session) SetWaite() {
	atomic.CompareAndSwapInt32(&session.waitFlag, 0, 1)
}

//判断连接是否为等待状态
func (session *Session) IsWaiting() bool {
	return atomic.LoadInt32(&session.waitFlag) == 1
}

func (session *Session) SetCodec(codec Codec) {
	session.codec = codec
}

func (session *Session) SetConnType(connType int8) {
	session.connType = connType
}

func (session *Session) GetConnType() int8 {
	return session.connType
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
	var shareKey []byte
	if !IsBytesAllZero(session.shareKey) {
		shareKey = session.shareKey
	} else {

	}
	return shareKey
}

func (session *Session) sendLoop() {
	defer session.Close()
	for {
		select {
		case msg := <-session.sendChan:
			//TODO 解析这个msg
			buf, err := session.codec.Packet(msg, session)
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
		session.closeWait.Wait()
		err := session.conn.Close()
		close(session.closeChan)
		if session.manager != nil {
			session.manager.delSession(session)
		}
		return err
	}
	return SessionClosedErr
}

func (session *Session) IsHttp() bool {
	if session.cfg.ReadDeadLine > 0 {
		deadTime := time.Now().Add(time.Second * time.Duration(session.cfg.ReadDeadLine))
		session.conn.SetReadDeadline(deadTime)
	}
	b := IsHttp(session.r)
	if session.cfg.ReadDeadLine > 0 {
		session.conn.SetReadDeadline(time.Time{})
	}
	return b
}

func (session *Session) InitCodec() error {
	if session.IsHttp() {
		headers := GetHeader(session.r, session.cfg.MaxMsgSize)
		if IsWsHandshake(headers) {
			if err := CheckUpgrade(headers); err != nil {
				return err
			}
			acceptKey := ComputeAcceptedKey(headers["Sec-WebSocket-Key"])
			resp := CreateUpgradeResp(acceptKey)
			if err := session.Write([]byte(resp)); err != nil {
				return err
			}
			session.SetConnType(WS)
			session.SetCodec(ProtoWs)
		} else {
			session.SetConnType(HTTP)
			session.SetCodec(ProtoHttp)
		}
	} else {
		session.SetConnType(TCP)
		session.SetCodec(ProtoTcp)
	}
	return nil
}

func (session *Session) Receive() (buf []byte, err error) {
	return session.codec.UnPack(session)
}

func (session *Session) Write(buf []byte) (err error) {
	if session.cfg.WriteDeadLine > 0 {
		deadTime := time.Now().Add(time.Second * time.Duration(session.cfg.WriteDeadLine))
		session.conn.SetWriteDeadline(deadTime)
	}
	var onceWriteLen, writtenLen, totalLen = 0, 0, len(buf)
	for writtenLen < totalLen {
		onceWriteLen, err = session.conn.Write(buf[writtenLen:])
		if err != nil {
			logger.Debug("session write: ", zap.Error(err))
			return
		}
		writtenLen += onceWriteLen
	}
	if session.cfg.WriteDeadLine > 0 {
		session.conn.SetWriteDeadline(time.Time{})
	}
	return nil
}

func (session *Session) Send(msg interface{}) error {
	if session.IsClosed() {
		return SessionClosedErr
	}
	if session.sendChan == nil {
		//TODO 解析这个msg
		buf, err := session.codec.Packet(msg, session)
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
