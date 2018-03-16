package net_lib

import (
	"sync"
	"net"
)

const sessionMapNum = 32

type Manager struct {
	sessionMaps [sessionMapNum]sessionMap
	disposeFlag bool
	disposeOnce sync.Once
	disposeWait sync.WaitGroup
}

type sessionMap struct {
	sessions map[uint64]*Session
	sync.RWMutex
}

func NewManager() *Manager {
	manager := &Manager{}
	for i := 0; i < sessionMapNum; i++ {
		manager.sessionMaps[i].sessions = make(map[uint64]*Session)
	}
	return manager
}

func (manager *Manager) NewSession(conn net.Conn, codec Codec, sendChanSize int) *Session {
	session := newSession(manager, conn, codec, sendChanSize)
	manager.putSession(session)
	return session
}

func (manager *Manager) Dispose() {
	manager.disposeOnce.Do(func() {
		manager.disposeFlag = true
		for i := 0; i < sessionMapNum; i++ {
			smap := manager.sessionMaps[i]
			smap.Lock()
			for _, session := range smap.sessions {
				session.Close()
			}
			smap.Unlock()
		}
		manager.disposeWait.Wait()
	})
}

func (manager *Manager) GetSession(sessionId uint64) *Session {
	smap := &manager.sessionMaps[sessionId%sessionMapNum]
	smap.RLock()
	defer smap.RUnlock()
	session, _ := smap.sessions[sessionId]
	return session
}

func (manager *Manager) putSession(session *Session) {
	smap := manager.sessionMaps[session.id % sessionMapNum]
	smap.Lock()
	defer smap.Unlock()
	smap.sessions[session.id] = session
	manager.disposeWait.Add(1)
}

func (manager *Manager) delSession(session *Session) {
	if manager.disposeFlag == true {
		manager.disposeWait.Done()
	}
	smap := manager.sessionMaps[session.id % sessionMapNum]
	smap.Lock()
	defer smap.Unlock()
	delete(smap.sessions, session.id)
	manager.disposeWait.Done()
}
