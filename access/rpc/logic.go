package rpc

import (
	"google.golang.org/grpc"
	//"github.com/imkuqin-zw/ZWChat/etcd"
	//"github.com/golang/glog"
	//"github.com/imkuqin-zw/ZWChat/access/config"
)

type LogicRPCCli struct {
	conn *grpc.ClientConn
}

func NewLogicRPCCli() (logicRPCCli *LogicRPCCli, err error) {
	//r := etcd.NewResolver(config.Conf.RpcClient.LoginClient.ServerName)
	//b := grpc.RoundRobin(r)
	//conn, err := grpc.Dial(config.Conf.RpcClient.LoginClient.Target, grpc.WithInsecure(), grpc.WithBalancer(b))
	//if err != nil {
	//	glog.Error(err)
	//	panic(err)
	//}
	//logicRPCCli = &LogicRPCCli{
	//	conn: conn,
	//}
	return
}