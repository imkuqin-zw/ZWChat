package net_lib

import "time"

type ProtoWsCode struct{}

func (codec *ProtoWsCode) Packet(msg interface{}, session *Session) ([]byte, error) {
	authKeyId := session.GetShareKeyId()
	shareKey := session.GetShareKey(authKeyId)
	body, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		logger.Error("Proto Packet Marshal err: ", zap.Error(err))
		return nil, err
	}
	result := new(Writer)
	if len(shareKey) == 0 { // 不加密
		result.WriteUint32(24 + uint32(len(body)))
		result.Write(authKeyId, make([]byte, 16), body)
	} else { // 加密
		msgKey, enBytes, err := codec.encrypt(shareKey, body)
		if err != nil {
			return nil, err
		}
		result.WriteUint32(24 + uint32(len(enBytes)))
		result.Write(authKeyId, msgKey, enBytes)
	}
	return result.Bytes(), nil
}

func (codec *ProtoWsCode) UnPack(session *Session) ([]byte, error) {
	if session.cfg.ReadDeadLine > 0 {
		deadTime := time.Now().Add(time.Second * time.Duration(session.cfg.ReadDeadLine))
		session.conn.SetReadDeadline(deadTime)
	}

	if session.cfg.ReadDeadLine > 0 {
		session.conn.SetReadDeadline(time.Time{})
	}

	for session.wsConn.readErr == nil {
		frameType, err := session.advanceFrame()
		if err != nil {
			c.readErr = hideTempErr(err)
			break
		}
	}



	return
}