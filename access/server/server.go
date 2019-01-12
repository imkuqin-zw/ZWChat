package server

import (
	"github.com/imkuqin-zw/ZWChat/access/client"
	"github.com/imkuqin-zw/ZWChat/access/rpc"
	"github.com/imkuqin-zw/ZWChat/lib/net_lib"
	"go.uber.org/zap"
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"fmt"
)

type Server struct {
	Server *net_lib.Server
}

func New() (s *Server) {
	s = &Server{}
	return
}

func (s *Server) Loop(rpcClient *rpc.RPCClient) {
	for {
		session, err := s.Server.Accept()
		if err != nil {
			logger.Error("session accept", zap.Error(err))
			continue
		}
		c := client.New(session, rpcClient)
		logger.Debug("session accept", zap.String("ip", session.RemoteIp),
			zap.String("port", session.RemotePort))
		go s.sessionLoop(c)
	}
}

func (s *Server) sessionLoop(client *client.Client) {
	if err := client.Session.InitCodec(); err != nil {
		logger.Error("Server SessionLoop", zap.Error(err))
		return
	}
	for {
		if client.Session.IsWaiting() {
			break
		}
		reqData, err := client.Session.Receive()
		if err != nil {
			logger.Error("session receive", zap.Error(err))
			return
		}
		fmt.Println(string(reqData))
		client.Session.Send(reqData)
		//if reqData != nil {
		//	baseCMD := &protobuf.Cmd{}
		//	if err = proto.Unmarshal(reqData, baseCMD); err != nil {
		//		if err = client.Session.Send(&protobuf.Error{
		//			Cmd:     baseCMD.Cmd,
		//			ErrCoed: ecode.ServerErr.Uint32(),
		//			ErrMsg:  ecode.ServerErr.String(),
		//		}); err != nil {
		//			glog.Error(err)
		//		}
		//		continue
		//	}
		//	if err = client.Parmse(baseCMD.Cmd, reqData); err != nil {
		//		glog.Error(err)
		//		continue
		//	}
		//}
	}
}
