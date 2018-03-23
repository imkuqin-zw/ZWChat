package rpc

import "google.golang.org/grpc"

type RegisterRPCCli struct {
	conn *grpc.ClientConn
}

func NewRegisterRPCCli()  (registerRPCCli *RegisterRPCCli, err error) {
	return
}