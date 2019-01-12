package net_lib

import (
	"io"
	"net"
	"strings"
	"time"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
)

type Server struct {
	manager         *Manager
	listener        net.Listener
	defaultCode     Codec
	sendChannelSize int
	sessionCfg      *SessionCfg
}

func NewServer(l net.Listener, sendChannelSize int, cfg *SessionCfg) *Server {
	return &Server{
		listener:        l,
		manager:         NewManager(),
		defaultCode:     ProtoTcp,
		sendChannelSize: sendChannelSize,
		sessionCfg:      cfg,
	}
}

func (server *Server) Listener() net.Listener {
	return server.listener
}

func (server *Server) Accept() (*Session, error) {
	var tempDelay time.Duration
	for {
		conn, err := server.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if tempDelay > time.Second {
					tempDelay = time.Second
				}
				time.Sleep(tempDelay)
				continue
			}
			// TODO 可能需要优化一下，但现在技术有限
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil, io.EOF
			}
			return nil, err
		}
		return server.manager.NewSession(conn, server.defaultCode, server.sendChannelSize, *server.sessionCfg), nil
	}
}

func (server *Server) Stop() {
	server.listener.Close()
	server.manager.Dispose()
}

func Serve(network, address string, cfg *SessionCfg, sendChanSize int) (*Server, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		logger.Fatal("Serve", zap.Error(err))
		return nil, err
	}
	return NewServer(listener, sendChanSize, cfg), nil
}
