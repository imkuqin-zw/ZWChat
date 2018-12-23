package net_lib

import (
	"net/http"
	"src(复件)/src(复件)/github.com/kataras/go-errors"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"time"
)

const http200Upgrade = "HTTP/1.1 101 Switching Protocols\r\n" +
	"Upgrade: websocket\r\n" +
	"Connection: Upgrade\r\n" +
	"Sec-WebSocket-Accept: %s\r\n" +
	"Date:%s\r\n\r\n"
var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

func IsWs(header http.Header) bool {
	if header.Get("Connection") == "Upgrade" {
		if header.Get("Upgrade") == "websocket" {
			return true
		}
	}
	return false
}

func CheckUpgrade(header http.Header) error {
	if header.Get("Sec-WebSocket-Key") == "" {
		return errors.New("Sec-WebSocket-Key is empty")
	}
	if header.Get("Sec-WebSocket-Version") != "13" {
		return errors.New("Sec-WebSocket-Version is not equate 13")
	}
	return nil
}

func ComputeAcceptedKey(header http.Header) string {
	SecWsKey := header.Get("Sec-WebSocket-Key")
	h := sha1.New()
	h.Write([]byte(SecWsKey))
	h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func CreateUpgradeResp(secWsKey string) string {
	TimeFormat := "Mon, 02 Jan 2006 15:04:05 GMT"
	modtimeStr := time.Now().UTC().Format(TimeFormat)
	return fmt.Sprintf(http200Upgrade, secWsKey, modtimeStr)
}
