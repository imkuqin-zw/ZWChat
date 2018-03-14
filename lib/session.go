package lib

import (
	"net"
	"sync/atomic"
	"fmt"
)

var SessionClosedErr = fmt.Errorf("[session] Session Closed")

var globalSessionId uint64

type Session struct {
	id 			uint64
	manager 	*Manager
	conn 		net.Conn
	closeFlag	int32
	closeChan 	chan int
	sendChan	chan interface{}
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
			lens := len(msg.([]byte))
			buf := make([]byte, 5 + lens)
			buf[0] = '0'
			buf[1] = byte(uint32(lens))
			buf[2] = byte(uint32(lens) >> 8)
			buf[3] = byte(uint32(lens) >> 16)
			buf[4] = byte(uint32(lens) >> 24)
			copy(buf[5:], msg.([]byte))
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

func (session *Session) Receive() ([]byte, error) {
	var buf = make([]byte,5)
	_, err := session.conn.Read(buf[0:5])
	if err != nil {
		return nil, err
	}
	lens := int(uint32(buf[1]) | uint32(buf[2])<<8 | uint32(buf[3])<<16 | uint32(buf[4])<<24)
	buf = make([]byte, lens)
	_, err = session.conn.Read(buf[0:lens])
	if err != nil {
		return nil, err
	}
	return buf, nil
}