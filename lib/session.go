package lib

import (
	"net"
	"sync/atomic"
	"fmt"
	"io"
)

var SessionClosedErr = fmt.Errorf("[session] Session Closed")

var globalSessionId uint64

type Session struct {
	id uint64
	manager *Manager
	conn 	net.Conn
	closeFlag	int32
	closeChan chan int
	sendChan	chan []byte
}

func NewSession(conn *net.Conn, sendChanSize int) *Session {
	return newSession(nil, conn, sendChanSize)
}

func newSession(manager *Manager, conn net.Conn, sendChanSize int) *Session {
	session := &Session{
		id: atomic.AddUint64(&globalSessionId, 1),
		manager: manager,
		closeChan: make(chan int),
		conn: conn,
	}
	if sendChanSize > 0 {
		session.sendChan = make(chan []byte, sendChanSize)
		go session.sendLoop()
	}
	return session
}

func (session *Session) sendLoop() {
	defer session.Close()
	for {
		select {
		case msg := <-session.sendChan:
			if _, err := session.conn.Write(msg); err != nil {
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

func (session *Session) Receive() ([]byte, error) {
	//test := io.Reader(session.conn)
}