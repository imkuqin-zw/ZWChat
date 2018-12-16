package net_lib

import (
	"bytes"
	"encoding/binary"
	"math/big"
)

type Writer struct {
	buf bytes.Buffer
}

func NewWriter(in []byte) *Writer {
	w := new(Writer)
	w.Write(in)
	return w
}

func (w *Writer) Bytes() []byte {
	return w.buf.Bytes()
}

func (w *Writer) WriteByte(v byte) {
	w.buf.WriteByte(v)
}

func (w *Writer) WriteUint24(v uint32) {
	var b = make([]byte, 3)
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[1] = byte(v >> 16)
	w.buf.Write(b)
}

func (w *Writer) WriteUint32(v uint32) {
	var b = make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	w.buf.Write(b[:])
}

func (w *Writer) WriteUint64(v uint64) {
	var b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	w.buf.Write(b[:])
}

func (w *Writer) WriteCmd(v uint32) {
	w.WriteUint32(v)
}

func (w *Writer) Write(v ...[]byte) {
	for _, item := range v {
		w.buf.Write(item)
	}

}

func (w *Writer) ZeroPad(n int) {
	w.buf.Grow(n)
	for i := 0; i < n; i++ {
		w.buf.WriteByte(0)
	}
}

func (w *Writer) WriteStringLen(v int) int {
	if v < 0 {
		panic("negative len")
	}
	if v < 254 {
		w.WriteByte(byte(v))
		return PaddingOf(1 + v)
	} else {
		w.buf.Grow(4)
		w.WriteByte(254)
		w.WriteUint24(uint32(v))
		return PaddingOf(4 + v)
	}
}

func (w *Writer) WriteString(v []byte) {
	pad := w.WriteStringLen(len(v))
	w.Write(v)
	w.ZeroPad(pad)
}

func (w *Writer) WriteBigInt(v *big.Int) {
	b := v.Bytes()
	if len(b) == 0 {
		b = []byte{0}
	}
	w.WriteString(b)
}

func PaddingOf(len int) int {
	rem := len % 4
	if rem == 0 {
		return 0
	} else {
		return 4 - rem
	}
}
