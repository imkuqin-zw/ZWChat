package etcd

import (
	etcdv3 "github.com/coreos/etcd/clientv3"
	"time"
	"fmt"
	"strings"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

var Prefix = "ZW"
var client *etcdv3.Client
var serviceKey string
var stopSignal = make(chan bool, 1)
var DialTimeout = time.Second

func Register(name, addr, target string, interval time.Duration, ttl time.Duration) (err error) {
	serviceKey = fmt.Sprintf("/%s/%s/%s", Prefix, name, addr)

	client, err = etcdv3.New(etcdv3.Config{
		Endpoints:strings.Split(target, ","),
		DialTimeout: DialTimeout,
	})
	if err != nil {
		glog.Error(err)
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		for {
			resp, _ := client.Grant(context.TODO(), int64(ttl/time.Second))
			if _, err := client.Get(context.Background(), serviceKey); err != nil {
				if err == rpctypes.ErrKeyNotFound {
					if _, err := client.Put(context.TODO(), serviceKey, addr, etcdv3.WithLease(resp.ID)); err != nil {
						glog.Error(err)
					}
				} else {
					glog.Error(err)
				}
			} else {
				if _, err := client.Put(context.Background(), serviceKey, addr, etcdv3.WithLease(resp.ID)); err != nil {
					glog.Error(err)
				}
			}
		}
		select {
		case <- stopSignal:
			return
		case <-ticker.C:
		}
	}()

	return nil
}

func UnRegister() (err error) {
	stopSignal <- true
	stopSignal = make(chan bool, 1)
	if _, err = client.Delete(context.Background(), serviceKey); err != nil {
		glog.Error(err)
	} else {
		glog.Infof("grpclb: deregister '%s' ok.")
	}
	return
}