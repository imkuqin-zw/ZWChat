package main

import (
	"github.com/imkuqin-zw/ZWChat/access/server"

	"github.com/imkuqin-zw/ZWChat/access/config"
	"flag"
	"github.com/golang/glog"
	"github.com/imkuqin-zw/ZWChat/lib/net_lib"
)

func main()  {
	var err error
	flag.Parse()
	if err = config.Init(); err != nil {
		glog.Error("config.Init error", err)
		panic(err)
	}
	accessServer := server.New()
	accessServer.Server, err = net_lib.Serve(config.Config.Server.Proto, config.Config.Server.Addr, &net_lib.ProtobufCodec{}, 0)
	if err != nil {
		glog.Error(err)
		panic(err)
	}
	accessServer.
}