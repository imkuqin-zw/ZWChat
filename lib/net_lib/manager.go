package net_lib

import (
	"net"
	"sync"
)

const sessionMapNum = 32

type Manager struct {
	sessionMaps      [sessionMapNum]sessionMap      //未登录的连接
	loginSessionMaps [sessionMapNum]loginSessionMap //登陆后的连接
	disposeFlag      bool
	disposeOnce      sync.Once
	disposeWait      sync.WaitGroup
}
type loginSessionMap struct {
	sessions map[uint64]map[int8]*Session
	sync.RWMutex
}

type sessionMap struct {
	sessions map[uint64]*Session
	sync.RWMutex
}

func NewManager() *Manager {
	manager := &Manager{}
	for i := 0; i < sessionMapNum; i++ {
		manager.sessionMaps[i].sessions = make(map[uint64]*Session)
		manager.loginSessionMaps[i].sessions = make(map[uint64]map[int8]*Session)
	}
	return manager
}

func (manager *Manager) NewSession(conn net.Conn, defaultCodec Codec, sendChanSize int, cfg SessionCfg) *Session {
	session := newSession(manager, conn, defaultCodec, sendChanSize, cfg)
	manager.putSession(session)
	return session
}

func (manager *Manager) Dispose() {
	manager.disposeOnce.Do(func() {
		manager.disposeFlag = true
		for i := 0; i < sessionMapNum; i++ {
			smap := &manager.sessionMaps[i]
			smap.Lock()
			for _, session := range smap.sessions {
				session.Close()
			}
			smap.Unlock()
			lsMap := &manager.loginSessionMaps[i]
			lsMap.Lock()
			for _, userSessionMap := range lsMap.sessions {
				for _, session := range userSessionMap {
					session.Close()
				}
			}
			lsMap.Unlock()
		}
		manager.disposeWait.Wait()
	})
}

func (manager *Manager) GetSessionByConnId(sessionId uint64) *Session {
	smap := &manager.sessionMaps[sessionId%sessionMapNum]
	smap.RLock()
	defer smap.RUnlock()
	session, _ := smap.sessions[sessionId]
	return session
}

func (manager *Manager) putSession(session *Session) {
	smap := &manager.sessionMaps[session.id%sessionMapNum]
	smap.Lock()
	defer smap.Unlock()
	smap.sessions[session.id] = session
	manager.disposeWait.Add(1)
}

func (manager *Manager) GetSessionMapByUid(uid uint64) map[int8]*Session {
	smap := &manager.loginSessionMaps[uid%sessionMapNum]
	smap.RLock()
	defer smap.RUnlock()
	session, _ := smap.sessions[uid]
	return session
}

func (manager *Manager) putLoginSession(session *Session) {
	smap := &manager.sessionMaps[session.id%sessionMapNum]
	smap.Lock()
	defer smap.Unlock()
	delete(smap.sessions, session.id)
	lsmap := &manager.loginSessionMaps[session.userId%sessionMapNum]
	lsmap.Lock()
	defer lsmap.Unlock()
	if lsmap.sessions[session.userId] == nil {
		userSessionMap :=  map[int8]*Session{session.connType: session}
		lsmap.sessions[session.userId] = userSessionMap
	} else {
		lsmap.sessions[session.userId][session.connType] = session
	}
	manager.disposeWait.Add(1)
}

func (manager *Manager) delSession(session *Session) {
	if manager.disposeFlag == true {
		manager.disposeWait.Done()
		return
	}
	if session.userId == 0 {
		smap := &manager.sessionMaps[session.id%sessionMapNum]
		smap.Lock()
		defer smap.Unlock()
		delete(smap.sessions, session.id)
	} else {
		smap := &manager.loginSessionMaps[session.userId%sessionMapNum]
		smap.Lock()
		defer smap.Unlock()
		delete(smap.sessions, session.userId)
	}
	manager.disposeWait.Done()
}
