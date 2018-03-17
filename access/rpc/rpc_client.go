package rpc

import "github.com/golang/glog"

type RPCClient struct {
	Logic *LogicRPCCli
}

func NewRPCClient() (c *RPCClient, err error) {
	logic, err := NewLogicRPCCli()
	if err != nil {
		glog.Error(err)
		return
	}
	c = &RPCClient{
		Logic: logic,
	}
	return
}