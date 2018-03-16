package server

import (
	"github.com/imkuqin-zw/ZWChat/lib/net_lib"
	"github.com/golang/glog"
)

type Server struct {
	Server *net_lib.Server
}

func New() (s *Server) {
	s = &Server{}
	return
}

func (s Server) Loop() {
	for {
		session, err := s.Server.Accept()
		if err != nil {
			glog.Error(err)
			continue
		}
		go s.sessionLoop(c)
	}
}