package config

import (
	commconf "github.com/imkuqin-zw/ZWChat/common/config"
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"os"
	"go.uber.org/zap"
	"github.com/imkuqin-zw/ZWChat/lib/net_lib"
)

var (
	confPath string
	Conf     *Config
)

type Config struct {
	Server           *commconf.Server                 `yaml:"server"`
	Path             *commconf.Path                   `yaml:"path"`
	ServiceDiscovery *commconf.ServiceDiscoveryServer `yaml:"serviceDiscovery"`
	RpcClient        *RpcClient                       `yaml:"rpcClient"`
	RpcServer        *commconf.Server                 `yaml:"rpcServer"`
	Etcd             *commconf.Etcd                   `yaml:"etcd"`
	Log              *zap.Config                      `yaml:"log"`
	SessionCfg       *net_lib.SessionCfg              `yaml:"sessionCfg"`
}

type RpcClient struct {
	LoginClient *commconf.ServiceDiscoveryClient `yaml:"loginClient"`
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
	return
}
