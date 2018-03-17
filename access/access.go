package main

import (
	"github.com/imkuqin-zw/ZWChat/access/server"

	"github.com/imkuqin-zw/ZWChat/access/config"
	"flag"
	"github.com/golang/glog"
	"github.com/imkuqin-zw/ZWChat/lib/net_lib"
	"github.com/imkuqin-zw/ZWChat/access/rpc"
)

func main()  {
	var err error
	flag.Parse()
	if err = config.Init(); err != nil {
		glog.Error("config.Init error", err)
		panic(err)
	}
	accessServer := server.New()
	accessServer.Server, err = net_lib.Serve(config.Conf.Server.Proto, config.Conf.Server.Addr, &net_lib.ProtobufCodec{}, 0)
	if err != nil {
		glog.Error(err)
		panic(err)
	}
	rpcClient, err := rpc.NewRPCClient()
	if err != nil {
		glog.Error(err)
		panic(err)
	}
	glog.Infof("%v %v", config.Conf.Server.Proto, config.Conf.Server.Addr)
	accessServer.Loop(rpcClient)
}

func init()  {
	flag.Set("alsologtostderr", "true")
	flag.Set("log_dir", "false")
}