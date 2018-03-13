package lib

import "net"

type Server struct {
	manager	*Manager
	listener net.Listener
	sendChannelSize	int
}

func (server *Server) Listener() net.Listener {
	return server.listener
}


