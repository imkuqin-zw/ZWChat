package main

import (
	"github.com/imkuqin-zw/ZWChat/access/server"

	"github.com/imkuqin-zw/ZWChat/access/config"
	"flag"
	"github.com/golang/glog"
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
	accessServer.Server, err = net_lib.Serve(config.Conf.Server.Proto, config.Conf.Server.Addr, 1)
	if err != nil {
		glog.Error(err)
		panic(err)
	}
	rpcClient, err := rpc.NewRPCClient()
	if err != nil {
		glog.Error(err)
		panic(err)
	}
	logger.Info("server init success", zap.String("addr", config.Conf.Server.Addr))
	accessServer.Loop(rpcClient)
}

func init()  {
	if err := config.Init(); err != nil {
		glog.Error("config.Init error", err)
		panic(err)
	}
	etcd.DialTimeout = config.Conf.Etcd.DialTimeout
	etcd.Prefix = config.Conf.Etcd.Prefix
	logger.InitLogger(config.Conf.Log)
}