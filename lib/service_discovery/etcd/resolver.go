package etcd

import (
	"google.golang.org/grpc/naming"
	"fmt"
	"github.com/golang/glog"
	etcdv3 "github.com/coreos/etcd/clientv3"
	"strings"
)

type resolver struct {
	serviceName string
}

func NewResolver(serviceName string) *resolver {
	return &resolver{serviceName: serviceName}
}

func (re *resolver) Resolve(target string) (naming.Watcher, error) {
	var err error
	if re.serviceName == "" {
		err = fmt.Errorf("no service name provided")
		glog.Error(err)
		return nil, err
	}

	client, err := etcdv3.New(etcdv3.Config{
		Endpoints: strings.Split(target, ","),
		DialTimeout: DialTimeout,
	})
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &watcher{re: re, client: client}, nil
}