package config

import (
	commconf "github.com/imkuqin-zw/ZWChat/common/config"
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"os"
	"fmt"
	"go.uber.org/zap"
)

var (
	confPath string
	Conf     *Config
)

type Config struct {
	Server           *commconf.Server
	Path             *commconf.Path
	ServiceDiscovery *commconf.ServiceDiscoveryServer
	RpcClient        *RpcClient
	RpcServer        *commconf.Server
	Etcd             *commconf.Etcd
	Log              *zap.Config
}

type RpcClient struct {
	LoginClient *commconf.ServiceDiscoveryClient
}

func init() {
	flag.StringVar(&confPath, "conf", "./access.yaml", "config path")
}

func Init() (err error) {
	var configBody []byte
	configBody, err = ioutil.ReadFile(confPath)
	if err != nil {
		return
	}
	Conf = &Config{}
	if err = yaml.Unmarshal(configBody, Conf); err != nil {
		return
	}
	Conf.Path = &commconf.Path{}
	Conf.Path.Root, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return
	}
	fmt.Println(Conf.Etcd)
	fmt.Println(Conf.RpcClient)
	fmt.Println(Conf.ServiceDiscovery)
	return
}
