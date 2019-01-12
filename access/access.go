package main

import (
	"github.com/imkuqin-zw/ZWChat/access/server"

	"github.com/imkuqin-zw/ZWChat/access/config"
	"flag"
	"github.com/imkuqin-zw/ZWChat/lib/net_lib"
	"github.com/imkuqin-zw/ZWChat/access/rpc"
	"github.com/imkuqin-zw/ZWChat/lib/service_discovery/etcd"
	"go.uber.org/zap"
	"github.com/imkuqin-zw/ZWChat/common/logger"
)

func main()  {
	var err error
	flag.Parse()
	accessServer := server.New()
	accessServer.Server, err = net_lib.Serve(config.Conf.Server.Proto, config.Conf.Server.Addr,
		config.Conf.SessionCfg, 1)
	if err != nil {
		return
	}
	rpcClient, err := rpc.NewRPCClient()
	if err != nil {
		return
	}
	logger.Info("server init success", zap.String("addr", config.Conf.Server.Addr))
	accessServer.Loop(rpcClient)
}

func init()  {
	if err := config.Init(); err != nil {
		panic(err)
		return
	}
	etcd.DialTimeout = config.Conf.Etcd.DialTimeout
	etcd.Prefix = config.Conf.Etcd.Prefix
	logger.InitLogger(config.Conf.Log)
}