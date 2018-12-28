package net_lib

import (
	"src(复件)/src(复件)/github.com/kataras/go-errors"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"time"
	"strings"
	"bytes"
)

const http200Upgrade = "HTTP/1.1 101 Switching Protocols\r\n" +
	"Upgrade: websocket\r\n" +
	"Connection: Upgrade\r\n" +
	"Sec-WebSocket-Accept: %s\r\n" +
	"Date:%s\r\n\r\n"
var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
var WsSWKErr = errors.New("Sec-WebSocket-Key header is error")
var WsUpgErr = errors.New("Upgrade header is error")

// 检查握手协议是否正确
func CheckUpgrade(header map[string]string) error {
	if header["Sec-WebSocket-Key"] == "" {
		return errors.New("Sec-WebSocket-Key is empty")
	}
	if header["Sec-WebSocket-Version"] != "13" {
		return errors.New("Sec-WebSocket-Version is not equate 13")
	}
	return nil
}

//生成accept key
func ComputeAcceptedKey(secWsKey string) string {
	h := sha1.New()
	h.Write([]byte(secWsKey))
	h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func CreateUpgradeResp(secWsKey string) string {
	TimeFormat := "Mon, 02 Jan 2006 15:04:05 GMT"
	modtimeStr := time.Now().UTC().Format(TimeFormat)
	return fmt.Sprintf(http200Upgrade, secWsKey, modtimeStr)
}

func GetHeader(r *Reader, maxSize uint32) (map[string]string) {
	data, _ := r.Peek(r.Buffered(maxSize))
	data = bytes.SplitN(data, []byte("\r\n\r\n"), 1)[0]
	headerBytes := bytes.Split(data, []byte("\r\n"))[1:]
	headers := make(map[string]string, len(headerBytes))
	for _, item := range headers {
		header := strings.Split(string(item), ":")
		headers[header[0]] = header[1]
	}
	return headers
}

func IsWsHandshake(header map[string]string) bool {
	if upgrade, ok := header["Upgrade"]; !ok {
		if upgrade == "websocket" {
			return true
		}
	}
	return false
}

func Unpack(r *Reader) ([]byte, error) {
	r.ReadN()
}