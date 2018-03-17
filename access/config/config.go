package config

import (
	commconf "github.com/imkuqin-zw/ZWChat/common/config"
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

var (
	confPath string
	Conf     *Config
)

type Config struct {
	Server *commconf.Server
}

func init()  {
	flag.StringVar(&confPath, "conf", "./access.yaml", "config path")
}

func Init() (err error) {
	var configBody []byte
	configBody, err = ioutil.ReadFile(confPath)
	if err != nil {
		return
	}
	Conf = &Config{}
	err = yaml.Unmarshal(configBody, Conf)
	return
}