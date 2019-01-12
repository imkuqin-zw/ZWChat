package config

import "time"

type Server struct {
	Proto string
	Addr  string
}

type Path struct {
	Root string
}


type Etcd struct {
	DialTimeout time.Duration
	Prefix string
}

type ServiceDiscoveryClient struct {
	Target string
	ServerName string
}

type ServiceDiscoveryServer struct {
	Target string
	ServerName string
	RpcAddr	string
	Interval time.Duration
	TTL time.Duration
}