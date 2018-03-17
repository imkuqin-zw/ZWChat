package main

import (
	"os"
	"fmt"
	"net"
	"github.com/imkuqin-zw/ZWChat/protobuf"
	"github.com/golang/protobuf/proto"
)

func main()  {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":1200")
	checkError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	fmt.Println("listen：" + listener.Addr().String())
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
	os.Exit(0)
}

func handleClient(conn net.Conn)  {
	clientIp := conn.RemoteAddr().String()
	fmt.Println(clientIp)
	defer conn.Close()
	for {
		var buf = make([]byte,5)
		_, err := conn.Read(buf[0:5])
		if err != nil {
			return
		}
		lens := int(uint32(buf[1]) | uint32(buf[2])<<8 | uint32(buf[3])<<16 | uint32(buf[4])<<24)
		fmt.Println(lens)

		buf = make([]byte, lens)
		_, err = conn.Read(buf[0:lens])
		msg := &protobuf.OffsetP2PMsg{}
		if err := proto.Unmarshal(buf, msg); err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(msg)
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}