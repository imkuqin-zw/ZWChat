package rpc

import (
	"github.com/imkuqin-zw/ZWChat/common/logger"
	"go.uber.org/zap"
)

type RPCClient struct {
	Logic *LogicRPCCli
}

func NewRPCClient() (c *RPCClient, err error) {
	logic, err := NewLogicRPCCli()
	if err != nil {
		logger.Fatal("NewLogicRPCCli", zap.Error(err))
		return
	}
	c = &RPCClient{
		Logic: logic,
	}
	return
}