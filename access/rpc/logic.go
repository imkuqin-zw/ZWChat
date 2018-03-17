package rpc

import "google.golang.org/grpc"

type LogicRPCCli struct {
	conn *grpc.ClientConn
}

func NewLogicRPCCli() (logicRPCCli *LogicRPCCli, err error) {
	return
}