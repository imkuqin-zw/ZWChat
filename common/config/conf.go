package config

import (
	"time"
)

type Server struct {
	Proto string `yaml:"proto"`
	Addr  string `yaml:"addr"`
}

type Path struct {
	Root string `yaml:"root"`
}

type Etcd struct {
	DialTimeout time.Duration `yaml:"dialTimeout"`
	Prefix      string        `yaml:"prefix"`
}

type ServiceDiscoveryClient struct {
	Target     string `yaml:"target"`
	ServerName string `yaml:"serverName"`
}

type ServiceDiscoveryServer struct {
	Target     string        `yaml:"target"`
	ServerName string        `yaml:"serverName"`
	RpcAddr    string        `yaml:"rpcAddr"`
	Interval   time.Duration `yaml:"interval"`
	TTL        time.Duration `yaml:"ttl"`
}
