package net_lib

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unsafe"
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

func GetHeader(r *Reader, maxSize uint32) map[string]string {
	data, _ := r.Peek(r.Buffered(maxSize))
	data = bytes.SplitN(data, []byte("\r\n\r\n"), 2)[0]
	headerBytes := bytes.Split(data, []byte("\r\n"))[1:]
	headers := make(map[string]string, len(headerBytes))
	for _, item := range headerBytes {
		header := strings.Split(string(item), ":")
		headers[header[0]] = strings.TrimSpace(header[1])
	}
	return headers
}

func IsWsHandshake(header map[string]string) bool {
	if upgrade, ok := header["Upgrade"]; ok {
		if upgrade == "websocket" {
			return true
		}
	}
	return false
}

const wordSize = int(unsafe.Sizeof(uintptr(0)))

//解码
func maskBytes(key [4]byte, pos int, b []byte) int {
	// Mask one byte at a time for small buffers.
	if len(b) < 2*wordSize {
		for i := range b {
			b[i] ^= key[pos&3]
			pos++
		}
		return pos & 3
	}

	// Mask one byte at a time to word boundary.
	if n := int(uintptr(unsafe.Pointer(&b[0]))) % wordSize; n != 0 {
		n = wordSize - n
		for i := range b[:n] {
			b[i] ^= key[pos&3]
			pos++
		}
		b = b[n:]
	}

	// Create aligned word size key.
	var k [wordSize]byte
	for i := range k {
		k[i] = key[(pos+i)&3]
	}
	kw := *(*uintptr)(unsafe.Pointer(&k))

	// Mask one word at a time.
	n := (len(b) / wordSize) * wordSize
	for i := 0; i < n; i += wordSize {
		*(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(&b[0])) + uintptr(i))) ^= kw
	}

	// Mask one byte at a time for remaining bytes.
	b = b[n:]
	for i := range b {
		b[i] ^= key[pos&3]
		pos++
	}

	return pos & 3
}
