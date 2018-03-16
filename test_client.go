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
		Msg: "45646",
	}
	body, err := proto.Marshal(msg)
	checkError(err)
	lens := len(body)
	fmt.Println(lens)
	var buf = make([]byte, lens + 5)
	buf[0] = 'w'
	buf[1] = byte(uint32(lens))
	buf[2] = byte(uint32(lens) >> 8)
	buf[3] = byte(uint32(lens) >> 16)
	buf[4] = byte(uint32(lens) >> 24)
	copy(buf[5:], body)
	fmt.Println(len(buf))
	for i := 0; i < 10; i++ {
		_, err = conn.Write(buf)
		checkError(err)
	}
	os.Exit(0)
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\r\n", err.Error())
		os.Exit(1)
	}
}