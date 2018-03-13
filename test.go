package main

import (
	"net"
	"fmt"
	"os"
	"github.com/imkuqin-zw/ZWChat/protobuf"
	"github.com/micro/protobuf/proto"
)

func main() {
	addr, err := net.ResolveTCPAddr("tcp4", ":1200")
	checkError(err)
	conn, err := net.DialTCP("tcp4", nil, addr)
	checkError(err)
	defer conn.Close()
	msg := &protobuf.OffsetP2PMsg{
		SourceUID: 1,
		TargetUID: 2,
		MsgID: "12",
		Msg: "fdsaf",
	}
	body, err := proto.Marshal(msg)
	checkError(err)
	_, err = conn.Write(body)
	checkError(err)

	os.Exit(0)
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\r\n", err.Error())
		os.Exit(1)
	}
}