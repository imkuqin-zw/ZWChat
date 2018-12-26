package net_lib

import (
	"bufio"
	"encoding/binary"
)

type Reader struct {
	r   *bufio.Reader
}

func NewReader(reader *bufio.Reader) *Reader {

	return &Reader{r: reader}
}

func (r *Reader) Peek(n int) ([]byte, error) {
	return r.r.Peek(n)
}


func (r *Reader) Read(data []byte) error {
	readLen, tempNum, total := 0, 0, len(data)
	var err error
	for readLen < total && err == nil {
		tempNum, err = r.r.Read(data)
		readLen += tempNum
	}
	return nil
}

func (r *Reader) ReadN(n uint32) ([]byte, error) {
	result := make([]byte, n)
	if n == 0 {
		return result, nil
	}

	if err := r.Read(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Reader) ReadUint32() (uint32, error) {
	buf := make([]byte, 4)
	if err := r.Read(buf); err != nil {
		return 0, err
	}
	v := binary.LittleEndian.Uint32(buf[0:4])
	return v, nil
}
