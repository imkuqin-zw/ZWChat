package server

import "github.com/imkuqin-zw/ZWChat/lib"

type Server struct {
	Server *lib.Server
}

func New() (s *Server) {
	s = &Server{}
	return
}