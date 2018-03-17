package client

import (
	"github.com/imkuqin-zw/ZWChat/lib/net_lib"
	"github.com/imkuqin-zw/ZWChat/access/rpc"
)

type Client struct {
	Session *net_lib.Session
	rpcClient *rpc.RPCClient
}

func New(session *net_lib.Session, rpcClient *rpc.RPCClient) *Client {
	return &Client{
		Session: session,
		rpcClient: rpcClient,
	}
}

func (client *Client) Parse(cmd uint32, reqData []byte) (err error) {
	switch cmd {

	}
	return
}