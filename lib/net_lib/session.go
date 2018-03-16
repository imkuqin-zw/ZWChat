package net_lib

import (
	"net"
	"sync/atomic"
	"fmt"
	"github.com/golang/glog"
)

var SessionClosedErr = fmt.Errorf("[session] Session Closed")

var globalSessionId uint64

type Session struct {
	id 			uint64
	manager 	*Manager
	conn 		net.Conn
	codec 		Codec
	closeFlag	int32
	closeChan 	chan int
	sendChan	chan interface{}
}

func NewSession(conn *net.Conn, codec Codec, sendChanSize int) *Session {
	return newSession(nil, conn, codec, sendChanSize)
}

func newSession(manager *Manager, conn net.Conn, codec Codec, sendChanSize int) *Session {
	session := &Session{
		id: atomic.AddUint64(&globalSessionId, 1),
		manager: manager,
		closeChan: make(chan int),
		conn: conn,
		codec: codec,
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
			buf, err := session.codec.Packet(msg)
			if err != nil {
				glog.Error(err)
				return
			}
			if _, err := session.conn.Write(buf); err != nil {
				return
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
	buf, err = session.codec.UnPack(session.conn)
	return
}