package net_lib

import (
	"net/http"
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

func CheckUpgrade(header http.Header) error {
	if header.Get("Sec-WebSocket-Key") == "" {
		return errors.New("Sec-WebSocket-Key is empty")
	}
	if header.Get("Sec-WebSocket-Version") != "13" {
		return errors.New("Sec-WebSocket-Version is not equate 13")
	}
	return nil
}

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

func IsWs(r *Reader, maxSize uint32) (key string, b bool, err error) {
	data, _ := r.Peek(int(maxSize))
	data = bytes.SplitN(data, []byte("\r\n\r\n"), 1)[0]
	headers := bytes.Split(data, []byte("\r\n"))[1:]
	for _, item := range headers {
		header := strings.Split(string(item), ":")
		if header[0] == "Upgrade" {
			if len(header) == 2 {
				if "websocket" == strings.TrimSpace(header[1]) {
					b = true
					if key != "" {
						break
					}
				}
			} else {
				err = WsUpgErr
				return
			}
		} else if header[0] == "Sec-WebSocket-Key" {
			if len(header) == 2 {
				key = strings.TrimSpace(header[1])
				if b {
					break
				}
			} else {
				err = WsSWKErr
				return
			}
		}
	}
	return
}