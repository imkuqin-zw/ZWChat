package net_lib

import (
	"io"
	"errors"
)

func NewMessageReader(session *Session) io.Reader {
	return &messageReader{session: session}
}

type messageReader struct{ session *Session }

func (r *messageReader) Read(b []byte) (int, error) {
	s := r.session
	for s.wsConn.readErr == nil {
		if s.wsConn.readRemaining > 0 {
			if int64(len(b)) > s.wsConn.readRemaining {
				b = b[:s.wsConn.readRemaining]
			}
			n, err := s.r.Read(b)
			s.wsConn.readErr = err
			s.wsConn.readMaskPos = maskBytes(s.wsConn.readMaskKey, s.wsConn.readMaskPos, b[:n])
			s.wsConn.readRemaining -= int64(n)
			if s.wsConn.readRemaining > 0 && s.wsConn.readErr == io.EOF {
				s.wsConn.readErr = errUnexpectedEOF
			}
			return n, s.wsConn.readErr
		}

		if s.wsConn.readFinal {
			return 0, io.EOF
		}

		frameType, err := s.advanceFrame()
		switch {
		case err != nil:
			s.wsConn.readErr = err
		case frameType == TextMessage || frameType == BinaryMessage:
			s.wsConn.readErr = errors.New("websocket: internal error, unexpected text or binary in Reader")
		}
	}

	err := s.wsConn.readErr
	if err == io.EOF {
		err = errUnexpectedEOF
	}
	return 0, err
}

