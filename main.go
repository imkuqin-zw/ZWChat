package main

import (
	"os"
	"fmt"
	"net"
	"github.com/imkuqin-zw/ZWChat/protobuf"
	"github.com/micro/protobuf/proto"
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
		var body [2]byte
		n, err := conn.Read(body[0:])
		if err != nil {
			return
		}
		pData := body[:n]
		msg := &protobuf.OffsetP2PMsg{
			SourceUID: 1,
			TargetUID: 2,
			MsgID: "12",
			Msg: "fdsaf",
		}
		if err := proto.Unmarshal(pData, msg); err != nil {
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